//go:generate npm run export --silent --prefix ../ui
//go:generate go-bindata -pkg server -nocompress -o ./bindata_ui.go -prefix "../ui/out/" ../ui/out/...

package server

import (
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
	"github.com/infrahq/infra/internal/okta"
	"golang.org/x/crypto/acme/autocert"
)

type Options struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
	UI         bool
	UIProxy    bool
}

func Run(options Options) error {
	// Load configuration from file
	var contents []byte
	var err error
	if options.ConfigPath != "" {
		contents, err = ioutil.ReadFile(options.ConfigPath)
		if err != nil {
			fmt.Println("Could not open config file path: ", options.ConfigPath)
		}
	}

	// Initialize database
	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	// Import config file
	ImportConfig(db, contents)

	// Create a new Kubernetes instance
	kubernetes, err := NewKubernetes(db)
	if err != nil {
		fmt.Println("warning: could not connect to Kubernetes API", err)
	}

	go kubernetes.UpdatePermissions()

	sync := Sync{}
	sync.Start(func() {
		var providers []Provider

		if err := db.Not(&Provider{Kind: DefaultInfraProviderKind}).Find(&providers).Error; err != nil {
			fmt.Println(err)
		}

		for _, p := range providers {
			if p.Kind == "okta" {
				emails, err := okta.Emails(p.Domain, p.ClientID, p.ApiToken)
				if err != nil {
					fmt.Println(err)
				}

				err = p.SyncUsers(db, emails)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		kubernetes.UpdatePermissions()
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

	if options.UIProxy {
		remote, _ := url.Parse("http://localhost:3000")
		devProxy := httputil.NewSingleHostReverseProxy(remote)
		router.NoRoute(func(c *gin.Context) {
			devProxy.ServeHTTP(c.Writer, c.Request)
		})
	} else if options.UI {
		// Middleware to improve flashes of content if not logged in, and vice-versa if visiting /login with an existing valid token
		router.Use(func(c *gin.Context) {
			ext := filepath.Ext(c.Request.URL.Path)
			// Only redirect non-html files/pages
			if ext != "" && ext != ".html" {
				c.Next()
				return
			}

			// check token cookie
			// TODO(jmorganca): validate this cookie
			token, err := c.Cookie("token")

			// if there's no token
			if err != nil {
				var settings Settings
				err = db.First(&settings).Error
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "error"})
					return
				}

				// Check if there are no users -> redirect to signup
				var count int64
				db.Find(&User{}).Count(&count)
				if count == 0 && !settings.DisableSignup && !strings.HasPrefix(c.Request.URL.Path, "/signup") {
					c.Redirect(301, "/signup")
					return
				} else if count > 0 && !strings.HasPrefix(c.Request.URL.Path, "/login") {
					c.Redirect(301, "/login")
					return
				}
			}

			if token != "" && (strings.HasPrefix(c.Request.URL.Path, "/login") || strings.HasPrefix(c.Request.URL.Path, "/signup")) {
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

		router.NoRoute(func(c *gin.Context) {
			gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}})).ServeHTTP(c.Writer, c.Request)
		})
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
