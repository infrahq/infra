package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type config struct {
	Providers []struct {
	}
	Permissions []struct {
	}
}

// Run runs the infra server
func Server() error {
	// Load the config file
	// raw, _ := ioutil.ReadFile(options.ConfigPath)
	// if err != nil {
	// 	panic(err)
	// }

	// Extract YAML configuration
	// config := config{}
	// _ = yaml.Unmarshal(raw, &config)
	// if err != nil {
	// 	panic(err)
	// }

	// TODO: Parse OIDC config

	// TODO: start OIDC connect sync

	// TODO: load

	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Extract certificate information

	}

	router := httprouter.New()
	router.GET("/*all", handler)
	router.POST("/*all", handler)
	router.PUT("/*all", handler)
	router.PATCH("/*all", handler)
	router.DELETE("/*all", handler)

	fmt.Printf("Listening on port %v\n", 3001)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 3001), router))

	return nil
}
