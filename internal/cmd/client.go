package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/registry"
)

var ClientTimeoutDuration = 5 * time.Minute

func RunLocalClient() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	timer := time.NewTimer(ClientTimeoutDuration)
	go func() {
		<-timer.C
		os.Exit(0)
	}()

	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		components := strings.Split(r.URL.Path, "/")
		if len(components) < 3 {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}

		name := components[2]

		contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "destinations"))
		if err != nil {
			fmt.Println(err)
			return
		}

		var destinations []registry.Destination
		err = json.Unmarshal(contents, &destinations)
		if err != nil {
			http.Error(w, "could not read destinations from ~/.infra/destinations", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		var destination registry.Destination
		for _, d := range destinations {
			if d.Name == name {
				destination = d
			}
		}

		if destination == (registry.Destination{}) {
			http.Error(w, "path not found", http.StatusNotFound)
			fmt.Println(err)
			return
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

		r.Header.Add("X-Infra-Authorization", r.Header.Get("Authorization"))
		r.Header.Del("Authorization")

		http.StripPrefix("/client/"+name, proxy).ServeHTTP(w, r)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/", proxyHandler)

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
		Handler:   mux,
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
