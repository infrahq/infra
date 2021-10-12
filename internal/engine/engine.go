package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry"
	"github.com/infrahq/infra/internal/timer"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Options struct {
	Registry       string
	Name           string
	ForceTLSVerify bool
	APIKey         string
	TLSCache       string
}

type jwkCache struct {
	mu          sync.Mutex
	key         *jose.JSONWebKey
	lastChecked time.Time

	client  *http.Client
	baseURL string
}

func (j *jwkCache) getjwk() (*jose.JSONWebKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.lastChecked != (time.Time{}) && time.Now().Before(j.lastChecked.Add(JWKCacheRefresh)) {
		return j.key, nil
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, j.baseURL+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, err
	}

	res, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}

	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	if len(response.Keys) < 1 {
		return nil, errors.New("no jwks provided by registry")
	}

	j.lastChecked = time.Now()
	j.key = &response.Keys[0]

	return &response.Keys[0], nil
}

var JWKCacheRefresh = 5 * time.Minute

type GetJWKFunc func() (*jose.JSONWebKey, error)

type HttpContextKeyEmail struct{}

func jwtMiddleware(destination string, getjwk GetJWKFunc, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		raw := strings.ReplaceAll(authorization, "Bearer ", "")
		if raw == "" {
			logging.L.Debug("No bearer token found")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		tok, err := jwt.ParseSigned(raw)
		if err != nil {
			logging.L.Debug("Invalid jwt signature")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		key, err := getjwk()
		if err != nil {
			logging.L.Debug("Could not get jwk")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		out := make(map[string]interface{})
		claims := struct {
			jwt.Claims
			registry.CustomJWTClaims
		}{}
		if err := tok.Claims(key, &claims, &out); err != nil {
			logging.L.Debug("Invalid token claims")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		err = claims.Claims.Validate(jwt.Expected{
			Issuer: "infra",
			Time:   time.Now(),
		})
		switch {
		case errors.Is(err, jwt.ErrExpired):
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		case err != nil:
			logging.L.Debug("Invalid JWT")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := validator.New().Struct(claims.CustomJWTClaims); err != nil {
			logging.L.Debug("JWT custom claims not valid")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if claims.Destination != destination {
			logging.L.Sugar().Debugf("JWT custom claims destination %q does not match expected destination %q", claims.Destination, destination)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, HttpContextKeyEmail{}, claims.Email)
		next(w, r.WithContext(ctx))
	})
}

func proxyHandler(ca []byte, bearerToken string, remote *url.URL) (http.HandlerFunc, error) {
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(ca)

	if !ok {
		return nil, errors.New("could not append ca to client cert bundle")
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(HttpContextKeyEmail{}).(string)
		if !ok {
			logging.L.Debug("Proxy handler unable to retrieve email from context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		r.Header.Set("Impersonate-User", email)
		r.Header.Set("Authorization", "Bearer "+bearerToken)
		http.StripPrefix("/proxy", proxy).ServeHTTP(w, r)
	}, nil
}

type BearerTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (b *BearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.Token != "" {
		req.Header.Set("Authorization", "Bearer "+b.Token)
	}

	return b.Transport.RoundTrip(req)
}

func Run(options Options) error {
	registryTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if !options.ForceTLSVerify {
		// TODO (https://github.com/infrahq/infra/issues/174)
		// Find a way to re-use the built-in TLS verification code vs
		// this custom code based on the official go TLS example code
		// which states this is approximately the same.
		registryTLSConfig.InsecureSkipVerify = true
		registryTLSConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				DNSName:       cs.ServerName,
				Intermediates: x509.NewCertPool(),
			}

			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}

			_, err := cs.PeerCertificates[0].Verify(opts)
			if err != nil {
				logging.L.Warn("could not verify registry TLS certificates: " + err.Error())
			}

			return nil
		}
	}

	u, err := urlx.Parse(options.Registry)
	if err != nil {
		return err
	}

	u.Scheme = "https"

	ctx := context.WithValue(context.Background(), api.ContextServerVariables, map[string]string{"basePath": "v1"})
	ctx = context.WithValue(ctx, api.ContextAccessToken, options.APIKey)
	config := api.NewConfiguration()
	config.Host = u.Host
	config.Scheme = "https"
	config.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: registryTLSConfig,
		},
	}

	client := api.NewAPIClient(config)

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	name := options.Name
	if name == "" {
		name, err = k8s.Name()
		if err != nil {
			logging.L.Error(err.Error())
			return err
		}
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, func() string {
		endpoint, err := k8s.Endpoint()
		if err != nil {
			return ""
		}

		url, err := urlx.Parse(endpoint)
		if err != nil {
			return ""
		}

		return url.Hostname()
	})

	certCacheMiss := 0

	timer := timer.NewTimer()
	timer.Start(5*time.Second, func() {
		endpoint, err := k8s.Endpoint()
		if err != nil {
			logging.L.Error("endpoint: " + err.Error())
			return
		}

		url, err := urlx.Parse(endpoint)
		if err != nil {
			logging.L.Error("url parse: " + err.Error())
			return
		}

		caBytes, err := manager.Cache.Get(context.TODO(), fmt.Sprintf("%s.crt", url.Hostname()))
		if err != nil {
			if errors.Is(err, autocert.ErrCacheMiss) {
				// first attempt to get the certificate on new service start will
				// likely fail so a single cache miss is expected
				certCacheMiss++
				if certCacheMiss > 1 {
					logging.L.Error(err.Error())
					return
				}
			}

			logging.L.Error("cache get: " + err.Error())
			return
		}

		destination, _, err := client.DestinationsApi.CreateDestination(ctx).Body(api.DestinationCreateRequest{
			Name: name,
			Kubernetes: &api.DestinationKubernetes{
				Ca:       string(caBytes),
				Endpoint: endpoint,
			},
		}).Execute()
		if err != nil {
			logging.L.Error("Couldn't create destination: " + err.Error())
			return
		}

		roles, _, err := client.RolesApi.ListRoles(ctx).Destination(destination.Id).Execute()
		if err != nil {
			logging.L.Error("couldn't list roles: " + err.Error())
		}

		err = k8s.UpdateRoles(roles)
		if err != nil {
			logging.L.Error("couldn't update roles: " + err.Error())
			return
		}
	})

	defer timer.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			logging.L.Error(err.Error())
		}
	})

	remote, err := urlx.Parse(k8s.Config.Host)
	if err != nil {
		return fmt.Errorf("parsing host config: %w", err)
	}

	ca, err := ioutil.ReadFile(k8s.Config.TLSClientConfig.CAFile)
	if err != nil {
		return fmt.Errorf("reading CA file: %w", err)
	}

	ph, err := proxyHandler(ca, k8s.Config.BearerToken, remote)
	if err != nil {
		return fmt.Errorf("setting proxy handler: %w", err)
	}

	cache := jwkCache{
		client: &http.Client{
			Transport: &BearerTransport{
				Token: options.APIKey,
				Transport: &http.Transport{
					TLSClientConfig: registryTLSConfig,
				},
			},
		},
		baseURL: u.String(),
	}

	mux.Handle("/proxy/", jwtMiddleware(name, cache.getjwk, ph))

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   handlers.LoggingHandler(os.Stdout, mux),
	}

	logging.L.Info("serving on port 443")

	return tlsServer.ListenAndServeTLS("", "")
}
