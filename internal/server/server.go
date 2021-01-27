package server

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/rest"
)

// Run runs the infra server
func Run() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	url, err := url.Parse(config.Host)
	if err != nil {
		panic(err.Error())
	}

	transport, err := rest.TransportFor(config)
	if err != nil {
		panic(err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = transport

	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		proxy.ServeHTTP(w, r)
	}

	router := httprouter.New()
	router.GET("/*subpath", handler)
	router.POST("/*subpath", handler)
	router.PUT("/*subpath", handler)
	router.PATCH("/*subpath", handler)
	router.DELETE("/*subpath", handler)

	fmt.Printf("Listening on port 3090")
	log.Fatal(http.ListenAndServe(":3090", router))
}
