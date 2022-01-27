package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/pro/audit"
	"github.com/infrahq/infra/internal/timer"
	"github.com/infrahq/infra/uid"
)

type Options struct {
	Server        string   `yaml:"server"`
	Name          string   `yaml:"name"`
	Kind          string   `yaml:"kind"`
	APIToken      string   `yaml:"apiToken"`
	TLSCache      string   `yaml:"tlsCache"`
	SkipTLSVerify bool     `yaml:"skipTLSVerify"`
	Labels        []string `yaml:"labels"`
}

type jwkCache struct {
	mu          sync.Mutex
	key         *jose.JSONWebKey
	lastChecked time.Time

	client  *http.Client
	baseURL string
}

func (j *jwkCache) getJWK() (*jose.JSONWebKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.lastChecked != (time.Time{}) && time.Now().Before(j.lastChecked.Add(JWKCacheRefresh)) {
		return j.key, nil
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s/.well-known/jwks.json", j.baseURL), nil)
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
		return nil, errors.New("no jwks provided by infra")
	}

	j.lastChecked = time.Now()
	j.key = &response.Keys[0]

	return &response.Keys[0], nil
}

var JWKCacheRefresh = 5 * time.Minute

type getJWKFunc func() (*jose.JSONWebKey, error)

func jwtMiddleware(next http.Handler, destination string, destinationName string, getJWK getJWKFunc) http.Handler {
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

		key, err := getJWK()
		if err != nil {
			logging.L.Debug("Could not get jwk")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		out := make(map[string]interface{})
		claims := struct {
			jwt.Claims
			claims.Custom
		}{}
		if err := tok.Claims(key, &claims, &out); err != nil {
			logging.L.Debug("Invalid token claims")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		err = claims.Claims.Validate(jwt.Expected{
			Issuer: "InfraHQ",
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

		if err := validator.New().Struct(claims.Custom); err != nil {
			logging.L.Debug("JWT custom claims not valid")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if claims.Destination != destination {
			logging.S.Debugf("JWT custom claims destination %q does not match expected destination %q", claims.Destination, destination)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, internal.HttpContextKeyEmail{}, claims.Email)
		ctx = context.WithValue(ctx, internal.HttpContextKeyGroups{}, claims.Groups)
		ctx = context.WithValue(ctx, internal.HttpContextKeyDestination{}, destination)
		ctx = context.WithValue(ctx, internal.HttpContextKeyDestinationName{}, destinationName)
		next.ServeHTTP(w, r.WithContext(ctx))
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
		email, ok := r.Context().Value(internal.HttpContextKeyEmail{}).(string)
		if !ok {
			logging.L.Debug("Proxy handler unable to retrieve email from context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		groups, ok := r.Context().Value(internal.HttpContextKeyGroups{}).([]string)
		if !ok {
			logging.L.Debug("Proxy handler unable to retrieve groups from context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		r.Header.Set("Impersonate-User", fmt.Sprintf("infra:%s", email))

		for _, g := range groups {
			r.Header.Add("Impersonate-Group", fmt.Sprintf("infra:%s", g))
		}

		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
		proxy.ServeHTTP(w, r)
	}, nil
}

type BearerTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (b *BearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.Token))
	}

	return b.Transport.RoundTrip(req)
}

func Run(options Options) error {
	hostTLSConfig := &tls.Config{MinVersion: tls.VersionTLS12}

	if options.SkipTLSVerify {
		// TODO (https://github.com/infrahq/infra/issues/174)
		// Find a way to re-use the built-in TLS verification code vs
		// this custom code based on the official go TLS example code
		// which states this is approximately the same.
		hostTLSConfig.InsecureSkipVerify = true
		hostTLSConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				DNSName:       cs.ServerName,
				Intermediates: x509.NewCertPool(),
			}

			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}

			_, err := cs.PeerCertificates[0].Verify(opts)
			if err != nil {
				logging.S.Warnf("could not verify Infra TLS certificates: %s", err.Error())
			}

			return nil
		}
	}

	engineAPITokenURI, err := url.Parse(options.APIToken)
	if err != nil {
		return err
	}

	var engineAPIToken string

	switch engineAPITokenURI.Scheme {
	case "":
		// option does not have a scheme, assume it is plaintext
		engineAPIToken = string(options.APIToken)
	case "file":
		// option is a file path, read contents from the path
		contents, err := ioutil.ReadFile(engineAPITokenURI.Path)
		if err != nil {
			return err
		}

		engineAPIToken = string(contents)

	default:
		return fmt.Errorf("unsupported secret format %s", engineAPITokenURI.Scheme)
	}

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	u, err := urlx.Parse(options.Server)
	if err != nil {
		return err
	}

	u.Scheme = "https"

	client := api.Client{
		Url:   u.String(),
		Token: engineAPIToken,
		Http: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: hostTLSConfig,
			},
		},
	}

	name, chksm, err := k8s.Name()
	if err != nil {
		logging.S.Errorf("k8s error: %w", err)
		return err
	}

	if options.Name == "" {
		options.Name = name
	}

	kind := api.DestinationKind(options.Kind)
	if !kind.IsValid() {
		return fmt.Errorf("unknown destination kind: %s", options.Kind)
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, func() string {
		host, _, err := k8s.Endpoint()
		if err != nil {
			return ""
		}

		url, err := urlx.Parse(host)
		if err != nil {
			return ""
		}

		return url.Hostname()
	})

	var destinationID uid.ID

	timer := timer.NewTimer()
	timer.Start(5*time.Second, func() {
		if destinationID == 0 {
			host, port, err := k8s.Endpoint()
			if err != nil {
				logging.S.Errorf("endpoint: %w", err)
				return
			}

			logging.S.Debugf("endpoint is: %s:%d", host, port)

			if ipv4 := net.ParseIP(host); ipv4 == nil {
				// wait for DNS resolution if endpoint is not an IPv4 address
				if _, err := net.LookupIP(host); err != nil {
					logging.L.Error("endpoint DNS could not be resolved, waiting to register")
				}
			}

			endpoint := fmt.Sprintf("%s:%d", host, port)

			url, err := urlx.Parse(endpoint)
			if err != nil {
				logging.S.Errorf("url parse: %s", err.Error())
				return
			}

			caBytes, err := manager.Cache.Get(context.TODO(), fmt.Sprintf("%s.crt", url.Hostname()))
			if err != nil {
				if errors.Is(err, autocert.ErrCacheMiss) {
					return
				} else {
					logging.S.Errorf("cache get: %s", err.Error())
					return
				}
			}

			destinations, err := client.ListDestinations(chksm)
			if err != nil {
				logging.S.Errorf("error listing destinations: %w", err)
				return
			}

			switch len(destinations) {
			case 0:
				request := &api.CreateDestinationRequest{
					NodeID: chksm,
					Name:   options.Name,
					Kind:   kind,
					Labels: options.Labels,
					Kubernetes: &api.DestinationKubernetes{
						CA:       string(caBytes),
						Endpoint: endpoint,
					},
				}

				destination, err := client.CreateDestination(request)
				if err != nil {
					logging.S.Errorf("error creating destination: %w", err)
					return
				}

				destinationID = destination.ID
			case 1:
				request := &api.UpdateDestinationRequest{
					NodeID: chksm,
					Name:   options.Name,
					Kind:   kind,
					Labels: options.Labels,
					Kubernetes: &api.DestinationKubernetes{
						CA:       string(caBytes),
						Endpoint: endpoint,
					},
				}

				_, err := client.UpdateDestination(destinations[0].ID, request)
				if err != nil {
					logging.S.Errorf("error updating destination: %w", err)
				}
			default:
				// this shouldn't happen
				logging.L.Info("unexpected result from ListDestinations")
				return
			}
		}

		grants, err := client.ListGrants(api.DestinationKindKubernetes, destinationID)
		if err != nil {
			logging.S.Errorf("error listing grants: %w", err)
			return
		}

		err = k8s.UpdateRoles(grants)
		if err != nil {
			logging.S.Errorf("error updating grants: %w", err)
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
				Token: engineAPIToken,
				Transport: &http.Transport{
					TLSClientConfig: hostTLSConfig,
				},
			},
		},
		baseURL: u.String(),
	}

	mux.Handle("/proxy/", http.StripPrefix("/proxy", jwtMiddleware(audit.AuditMiddleware(ph), chksm, options.Name, cache.getJWK)))

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   handlers.CustomLoggingHandler(io.Discard, mux, logging.ZapLogFormatter),
		ErrorLog:  logging.StandardErrorLog(),
	}

	logging.L.Info("serving on port 443")

	return tlsServer.ListenAndServeTLS("", "")
}
