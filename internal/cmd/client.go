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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goware/urlx"
	v1 "github.com/infrahq/infra/internal/v1"
	"github.com/natefinch/lumberjack"
)

const (
	INFRA_HIDDEN_DIR = ".infra"
	CLIENT_DIR       = "client"
)

var (
	ClientTimeoutDuration = 5 * time.Minute
	errorLogger           log.Logger // writes errors from the proxy handler to a file
)

func RunLocalClient() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	errorLogger.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(homeDir, INFRA_HIDDEN_DIR, CLIENT_DIR, "proxy_error.log"),
		MaxSize:    1, // megabyte
		MaxBackups: 1,
	})
	errorLogger.SetPrefix(time.Now().Format("2006-01-02 15:04:05 "))

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

		contents, err := ioutil.ReadFile(filepath.Join(homeDir, INFRA_HIDDEN_DIR, "destinations"))
		if err != nil {
			fmt.Println(err)
			return
		}

		var destinations []v1.Destination
		err = json.Unmarshal(contents, &destinations)
		if err != nil {
			http.Error(w, "could not read destinations from ~/.infra/destinations", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}

		var destination *v1.Destination
		for i := range destinations {
			if destinations[i].Name == name {
				destination = &destinations[i]
			}
		}

		if destination == nil {
			fmt.Println("could not load destination information for destination " + name)
			return
		}

		var endpoint, ca, saToken string
		namespace := "default"

		if kube := destination.GetKubernetes(); kube != nil {
			endpoint = kube.Endpoint
			ca = kube.Ca
			namespace = kube.Namespace
			saToken = kube.SaToken
		}

		remote, err := urlx.Parse(endpoint + "/api/v1/namespaces/" + namespace + "/services/http:infra-engine:80/proxy/proxy")
		if err != nil {
			errorLogger.Println(err)
			return
		}

		remote.Scheme = "https"

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(ca))
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		timer.Reset(ClientTimeoutDuration)

		if r.Header.Get("Upgrade") != "" {
			r.Header.Add("X-Infra-Query", r.URL.RawQuery)
		}

		r.Header.Add("X-Infra-Authorization", r.Header.Get("Authorization"))
		r.Header.Set("Authorization", "Bearer "+saToken)

		http.StripPrefix("/client/"+name, proxy).ServeHTTP(w, r)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/client/", proxyHandler)

	certBytes, err := ioutil.ReadFile(filepath.Join(homeDir, INFRA_HIDDEN_DIR, CLIENT_DIR, "cert.pem"))
	if err != nil {
		return err
	}

	keyBytes, err := ioutil.ReadFile(filepath.Join(homeDir, INFRA_HIDDEN_DIR, CLIENT_DIR, "key.pem"))
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

	if err = ioutil.WriteFile(filepath.Join(homeDir, INFRA_HIDDEN_DIR, CLIENT_DIR, "pid"), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return err
	}

	return tlsServer.ServeTLS(l, "", "")
}
