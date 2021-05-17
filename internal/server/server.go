package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/acme/autocert"
)

type ServerOptions struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
}

var PermissionOrdering = []string{"view", "edit", "admin"}

func IsEqualOrHigherPermission(a string, b string) bool {
	indexa := 0
	indexb := 0

	for i, p := range PermissionOrdering {
		if a == p {
			indexa = i
		}

		if b == p {
			indexb = i
		}
	}

	return indexa >= indexb
}

// Gets users permissions from config, with a catch-all of view
// TODO (jmorganca): should this be nothing instead of view?
func PermissionForEmail(email string, cfg *Config) string {
	for _, p := range cfg.Permissions {
		if p.Email == email {
			return p.Permission
		}
	}

	// Default to view
	return "view"
}

func Run(options *ServerOptions) error {
	if options.DBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		options.DBPath = filepath.Join(homeDir, ".infra")
	}

	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	defer db.Close()

	kube, err := NewKubernetes()
	if err != nil {
		fmt.Println("warning: no kubernetes cluster detected.")
	}

	config, err := NewConfig(options.ConfigPath)
	if err != nil {
		return err
	}

	sync := Sync{}
	sync.Start(func() {
		if config.Providers.Okta.Domain != "" {
			emails, err := config.Providers.Okta.Emails()
			if err != nil {
				fmt.Println(err)
			}

			db.Update(func(tx *bolt.Tx) error {
				return SyncUsers(tx, emails, "okta")
			})
		}

		kube.UpdatePermissions(db, config)
	})
	defer sync.Stop()

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()

	// Skip auth when accessing infra engine over
	unixRouter.Use(func(c *gin.Context) {
		c.Set("skipauth", true)
	})

	if err = addRoutes(unixRouter, db, kube, config); err != nil {
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
	if err = addRoutes(router, db, kube, config); err != nil {
		return err
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	if options.TLSCache != "" {
		manager.Cache = autocert.DirCache(options.TLSCache)
	}

	tlsServer := &http.Server{
		Addr:      ":8443",
		TLSConfig: manager.TLSConfig(),
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
