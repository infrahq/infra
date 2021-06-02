//go:generate npm run export --silent --prefix ../ui
//go:generate go-bindata -pkg server -nocompress -o ./bindata_ui.go -prefix "../ui/out/" ../ui/out/...

package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
	"github.com/imdario/mergo"
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type Options struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
	UI         bool
	UIProxy    bool
}

func updatePermissions(config *Config, db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, p := range config.Permissions {
			var dbp Permission
			err := tx.Where(&Permission{Name: p.Name}).First(&dbp).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			// For now ignore references to permissions that don't exist
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}

			var usersToAssociate []User
			err = tx.Where("email IN ?", p.Users).Not(&User{PermissionID: dbp.ID}).Find(&usersToAssociate).Error
			if err != nil {
				return err
			}

			err = tx.Model(&dbp).Association("Users").Append(usersToAssociate)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func Run(options Options) error {
	// Load configuration from file
	var config Config
	if options.ConfigPath != "" {
		contents, err := ioutil.ReadFile(options.ConfigPath)
		if err != nil {
			fmt.Println("Could not open config file path: ", options.ConfigPath)
		} else {
			err = yaml.Unmarshal([]byte(contents), &config)
			if err != nil {
				return err
			}
		}
	}

	// Initialize database
	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	err = updatePermissions(&config, db)
	if err != nil {
		return err
	}

	var s Settings
	err = db.First(&s).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var dbConfig Config
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		yaml.Unmarshal(s.Config, &dbConfig)
	}

	// Migrate / initialize db configuration and save
	InitConfig(&dbConfig)

	bs, err := yaml.Marshal(dbConfig)
	if err != nil {
		return err
	}
	s.Config = bs
	db.Save(&s)

	// Create merged config from configuration file and database file
	if err := mergo.Merge(&dbConfig, &config); err != nil {
		return err
	}

	// Create a new Kubernetes instance
	kubernetes, err := NewKubernetes(db)
	if err != nil {
		fmt.Println("warning: could connect to Kubernetes API", err)
	}

	var cs ConfigStore
	cs.set(&dbConfig)

	sync := Sync{}
	sync.Start(func() {
		if cs.get().Providers.Okta.Domain != "" {
			emails, err := okta.Emails(cs.get().Providers.Okta.Domain, cs.get().Providers.Okta.ClientID, cs.get().Providers.Okta.ApiToken)
			if err != nil {
				fmt.Println(err)
			}

			err = SyncUsers(db, emails, "okta")
			if err != nil {
				fmt.Println(err)
			}

			err = updatePermissions(cs.get(), db)
			if err != nil {
				fmt.Println(err)
			}

			kubernetes.UpdatePermissions()
		}
	})

	defer sync.Stop()

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	unixRouter.Use(gin.Logger())
	unixRouter.Use(func(c *gin.Context) {
		c.Set("skipauth", true)
	})

	handlers := &Handlers{
		db:         db,
		cs:         &cs,
		kubernetes: kubernetes,
	}

	if err = handlers.addRoutes(unixRouter); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	os.Remove(filepath.Join(homeDir, ".infra", "infra.sock"))
	go func() {
		if err := unixRouter.RunUnix(filepath.Join(homeDir, ".infra", "infra.sock")); err != nil {
			log.Fatal(err)
		}
	}()

	router := gin.New()
	router.Use(gin.Logger())
	if err = handlers.addRoutes(router); err != nil {
		return err
	}

	if options.UI || options.UIProxy {
		// Login middleware for UI to avoid flash of content
		router.Use(func(c *gin.Context) {
			ext := filepath.Ext(c.Request.URL.Path)
			if ext != "" && ext != ".html" {
				c.Next()
				return
			}

			// check token cookie
			token, err := c.Cookie("token")

			if err != nil && !strings.HasPrefix(c.Request.URL.Path, "/login") {
				c.Redirect(301, "/login")
				return
			}

			if token != "" && strings.HasPrefix(c.Request.URL.Path, "/login") {
				keys, ok := c.Request.URL.Query()["next"]

				if !ok || len(keys[0]) < 1 {
					c.Redirect(301, "/")
				} else {
					c.Redirect(301, keys[0])
				}

				return
			}

			c.Next()
		})

		if options.UIProxy {
			remote, _ := url.Parse("http://localhost:3000")
			devProxy := httputil.NewSingleHostReverseProxy(remote)
			router.NoRoute(func(c *gin.Context) {
				devProxy.ServeHTTP(c.Writer, c.Request)
			})
		} else if options.UI {
			router.NoRoute(func(c *gin.Context) {
				gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}})).ServeHTTP(c.Writer, c.Request)
			})
		}
	}

	go router.Run(":80")

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(manager)

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
