package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/registry"
)

var ClientTimeoutDuration = 5 * time.Minute

func RunLocalClient() error {
	router := gin.New()
	router.Use(gin.Logger())

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	timer := time.NewTimer(ClientTimeoutDuration)
	go func() {
		<-timer.C
		os.Exit(0)
	}()

	proxyHandler := func(c *gin.Context) {
		type binds struct {
			Name string `uri:"name" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			log.Println(err)
			return
		}

		contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "destinations"))
		if err != nil {
			log.Println(err)
			return
		}

		var destinations []registry.Destination
		err = json.Unmarshal(contents, &destinations)
		if err != nil {
			log.Println(err)
			return
		}

		var destination registry.Destination
		for _, d := range destinations {
			if d.Name == params.Name {
				destination = d
			}
		}

		remote, err := url.Parse(destination.KubernetesEndpoint + "/api/v1/namespaces/infra/services/http:infra-engine:80/proxy/proxy")
		if err != nil {
			log.Println(err)
			return
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(destination.KubernetesCA))
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		timer.Reset(ClientTimeoutDuration)

		c.Request.Header.Add("X-Infra-Authorization", c.Request.Header.Get("Authorization"))
		c.Request.Header.Del("Authorization")

		http.StripPrefix("/client/"+params.Name, proxy).ServeHTTP(c.Writer, c.Request)
	}

	router.GET("/client/:name/*all", proxyHandler)
	router.POST("/client/:name/*all", proxyHandler)
	router.PUT("/client/:name/*all", proxyHandler)
	router.PATCH("/client/:name/*all", proxyHandler)
	router.DELETE("/client/:name/*all", proxyHandler)

	certBytes, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "client", "cert.pem"))
	if err != nil {
		return err
	}

	keyBytes, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "client", "key.pem"))
	if err != nil {
		return err
	}

	keypair, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{keypair}
	tlsServer := &http.Server{
		Addr:      "127.0.0.1:32710",
		TLSConfig: tlsConfig,
		Handler:   router,
	}

	l, err := net.Listen("tcp", "127.0.0.1:32710")
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "client", "pid"), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return err
	}

	return tlsServer.ServeTLS(l, "", "")
}
