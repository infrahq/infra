package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
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
	raw, _ := ioutil.ReadFile(options.ConfigPath)
	// if err != nil {
	// 	panic(err)
	// }

	config := config{}
	_ = yaml.Unmarshal(raw, &config)
	// if err != nil {
	// 	panic(err)
	// }

	// TODO: Parse OIDC config

	// TODO: start OIDC connect sync

	// TODO: load

	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

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
