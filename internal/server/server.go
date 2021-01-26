package server

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/rest"
)

// Run runs the infra server
func Run() {
	// Create the proxy
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Host:   strings.TrimPrefix(config.Host, "https://"),
		Scheme: "https",
	})

	handler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		transport, _ := rest.TransportFor(config)
		proxy.Transport = transport
		proxy.ServeHTTP(w, r)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	router := httprouter.New()
	router.GET("/*subpath", handler)

	fmt.Printf("Listening on port 3090")
	log.Fatal(http.ListenAndServe(":3090", router))
}
