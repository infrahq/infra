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
		fmt.Println("warning: could not connect to kubernetes api", err)
	}

	config, err := NewConfig(options.ConfigPath)
	if err != nil {
		fmt.Println("warning: could not open config file", err)
	}

	sync := Sync{}
	sync.Start(func() {
		if config.Providers.Okta.Domain != "" {
			emails, err := config.Providers.Okta.Emails()
			if err != nil {
				fmt.Println(err)
			}

			err = db.Update(func(tx *bolt.Tx) error {
				return SyncUsers(tx, emails, "okta")
			})
			if err != nil {
				fmt.Println(err)
			}
		}

		kube.UpdatePermissions(db, config)
	})
	defer sync.Stop()

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	unixRouter.Use(gin.Logger())
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
	router.Use(gin.Logger())
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
