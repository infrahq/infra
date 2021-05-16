package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/providers"
	"golang.org/x/crypto/acme/autocert"
)

type ServerOptions struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
}

func UpdatePermissions(data *data.Data, kube *kubernetes.Kubernetes) error {
	if data == nil {
		return errors.New("data cannot be nil")

	}

	users, err := data.ListUsers()
	if err != nil {
		return err
	}

	roleBindings := []kubernetes.RoleBinding{}
	for _, user := range users {
		roleBindings = append(roleBindings, kubernetes.RoleBinding{User: user.Email, Role: user.Permission})
	}

	return kube.UpdateRoleBindings(roleBindings)
}

func getSelfSignedOrLetsEncryptCert(certManager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// dirCache, ok := certManager.Cache.(autocert.DirCache)
		// if !ok {
		// 	dirCache = "certs"
		// }

		fmt.Println(hello.ServerName)
		fmt.Println(hello)

		// Try to use letsencrypt
		certificate, err := certManager.GetCertificate(hello)
		if err == nil {
			return certificate, nil
		}

		// Generate self-signed certificates
		fmt.Println("Falling back to self-signed ceritficate", err)
		return nil, errors.New("not implemented")
	}
}

func ServerRun(options *ServerOptions) error {
	if options.DBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		options.DBPath = filepath.Join(homeDir, ".infra")
	}

	data, err := data.NewData(options.DBPath)
	if err != nil {
		return err
	}

	defer data.Close()

	config, err := loadConfig(options.ConfigPath)
	if err != nil {
		return err
	}

	kube, err := kubernetes.NewKubernetes()
	if err != nil {
		fmt.Println("warning: no kubernetes cluster detected.")
	}

	okta := &providers.Okta{
		Domain:   config.Providers.Okta.Domain,
		ClientID: config.Providers.Okta.ClientID,
		ApiToken: config.Providers.Okta.ApiToken,
	}

	sync := Sync{}

	// TODO (jmorganca): fix error handling here
	sync.Start(func() {
		emails, _ := okta.Emails()
		syncUsers(data, "okta", emails)
		UpdatePermissions(data, kube)
	})

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	if err = addRoutes(unixRouter, data, kube, config, false); err != nil {
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
	if err = addRoutes(router, data, kube, config, true); err != nil {
		return err
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	if options.TLSCache != "" {
		manager.Cache = autocert.DirCache(options.TLSCache)
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(manager)

	tlsServer := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
