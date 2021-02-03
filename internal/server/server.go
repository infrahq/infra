package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"

	"k8s.io/client-go/rest"
)

// Options provides the configuration options for the Infra server
type Options struct {
	ConfigPath string
	Port       int
}

type config struct {
	Providers []struct {
		Kind   string
		Name   string
		Groups []string
		Users  []string
		Config struct {
			ClientID     string
			ClientSecret string
			IDToken      string
			IdpIssuerURL string
			RefreshToken string
		}
	}
}

// Run runs the infra server
func Run(options *Options) error {
	// Load the config file
	raw, err := ioutil.ReadFile(options.ConfigPath)
	if err != nil {
		return err
	}

	config := config{}
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return err
	}

	fmt.Printf("--- m:\n%v\n\n", config)

	// TODO: Parse OIDC config

	// TODO: start OIDC connect sync

	// TODO: load

	// Run proxy
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	url, err := url.Parse(kubeConfig.Host)
	if err != nil {
		panic(err.Error())
	}

	transport, err := rest.TransportFor(kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = transport

	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		proxy.ServeHTTP(w, r)
	}

	router := httprouter.New()
	router.GET("/*all", handler)
	router.POST("/*all", handler)
	router.PUT("/*all", handler)
	router.PATCH("/*all", handler)
	router.DELETE("/*all", handler)

	fmt.Printf("Listening on port %v\n", options.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", options.Port), router))

	return nil
}
