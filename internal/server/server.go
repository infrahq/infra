package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/providers"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v2"
)

type ServerOptions struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
}

type OktaConfig struct {
	Domain       string `yaml:"domain" json:"domain"`
	ClientID     string `yaml:"client-id" json:"client-id"`
	ClientSecret string `yaml:"client-secret"` // TODO(jmorganca): move this to a secret
	ApiToken     string `yaml:"api-token"`     // TODO(jmorganca): move this to a secret
}

type ServerConfig struct {
	Providers struct {
		Okta OktaConfig `yaml:"okta" json:"okta"`
	}
	Permissions []struct {
		User       string
		Group      string
		Permission string
	}
}

func loadConfig(path string) (*ServerConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return &ServerConfig{}, nil
	}

	if err != nil {
		return nil, err
	}

	var config ServerConfig
	err = yaml.Unmarshal([]byte(contents), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
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

	kubernetes, err := kubernetes.NewKubernetes()
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
	})

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	if err = addRoutes(unixRouter, data, kubernetes, config, false); err != nil {
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
	if err = addRoutes(router, data, kubernetes, config, true); err != nil {
		return err
	}

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	fmt.Println("Using certificate cache", options.TLSCache)

	if options.TLSCache != "" {
		m.Cache = autocert.DirCache(options.TLSCache)
	}

	tlsServer := &http.Server{
		Addr:      ":8443",
		TLSConfig: m.TLSConfig(),
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
