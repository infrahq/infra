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
	"github.com/infrahq/infra/internal/timer"
	"github.com/infrahq/infra/uid"

	rbacv1 "k8s.io/api/rbac/v1"
)

type Options struct {
	Server        string `yaml:"server"`
	Name          string `yaml:"name"`
	APIToken      string `yaml:"apiToken"`
	TLSCache      string `yaml:"tlsCache"`
	SkipTLSVerify bool   `yaml:"skipTLSVerify"`
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

func jwtMiddleware(next http.Handler, getJWK getJWKFunc) http.Handler {
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
			Time: time.Now(),
		})
		switch {
		case errors.Is(err, jwt.ErrExpired):
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		case err != nil:
			logging.S.Debugf("Invalid JWT %s", err.Error())
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := validator.New().Struct(claims.Custom); err != nil {
			logging.L.Debug("JWT custom claims not valid")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, internal.HttpContextKeyEmail{}, claims.Email)
		ctx = context.WithValue(ctx, internal.HttpContextKeyGroups{}, claims.Groups)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UpdateRoles converts infra grants to role-bindings in the current cluster
func updateRoles(c *api.Client, k *kubernetes.Kubernetes, grants []api.Grant) error {
	logging.L.Debug("syncing local grants from infra configuration")

	crSubjects := make(map[string][]rbacv1.Subject)                           // cluster-role: subject
	crnSubjects := make(map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) // cluster-role+namespace: subject

	for _, g := range grants {
		var name string
		var kind string

		switch {
		case strings.HasPrefix(g.Identity, "g:"):
			var id uid.ID

			err := id.UnmarshalText([]byte(strings.TrimPrefix(g.Identity, "g:")))
			if err != nil {
				return err
			}

			group, err := c.GetGroup(id)
			if err != nil {
				return err
			}

			name = group.Name
			kind = rbacv1.GroupKind

		case strings.HasPrefix(g.Identity, "u:"):
			var id uid.ID

			err := id.UnmarshalText([]byte(strings.TrimPrefix(g.Identity, "u:")))
			if err != nil {
				return err
			}

			user, err := c.GetUser(id)
			if err != nil {
				return err
			}

			name = user.Email
			kind = rbacv1.UserKind
		}

		subj := rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     kind,
			Name:     name,
		}

		parts := strings.Split(g.Resource, ".")

		var crn kubernetes.ClusterRoleNamespace
		switch len(parts) {
		// kubernetes.<cluster>
		case 2:
			crn.ClusterRole = g.Privilege
			crSubjects[g.Privilege] = append(crSubjects[g.Privilege], subj)

		// kubernetes.<cluster>.<namespace>
		case 3:
			crn.ClusterRole = g.Privilege
			crn.Namespace = parts[2]
			crnSubjects[crn] = append(crnSubjects[crn], subj)

		default:
			logging.S.Warnf("invalid grant resource: %s", g.Resource)
			continue
		}
	}

	if err := k.UpdateClusterRoleBindings(crSubjects); err != nil {
		return fmt.Errorf("update cluster role bindings: %w", err)
	}

	if err := k.UpdateRoleBindings(crnSubjects); err != nil {
		return fmt.Errorf("update cluster role bindings: %w", err)
	}

	return nil
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

		r.Header.Set("Impersonate-User", email)

		for _, g := range groups {
			r.Header.Add("Impersonate-Group", g)
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

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	autoname, chksm, err := k8s.Name()
	if err != nil {
		logging.S.Errorf("k8s name error: %w", err)
		return err
	}

	if options.Name == "" {
		options.Name = autoname
	}

	if !strings.HasPrefix(options.Name, "kubernetes.") {
		options.Name = fmt.Sprintf("kubernetes.%s", options.Name)
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

	u, err := urlx.Parse(options.Server)
	if err != nil {
		logging.S.Errorf("server: %w", err)
	}

	u.Scheme = "https"

	timer := timer.NewTimer()
	timer.Start(5*time.Second, func() {
		contents, err := ioutil.ReadFile(options.APIToken)
		if err != nil {
			logging.S.Errorf("could not load api token: %w", err)
			return
		}

		engineAPIToken := string(contents)

		client := &api.Client{
			Url:   u.String(),
			Token: engineAPIToken,
			Http: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: hostTLSConfig,
				},
			},
		}

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

			destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: "", UniqueID: chksm})
			if err != nil {
				logging.S.Errorf("error listing destinations: %w", err)
				return
			}

			switch len(destinations) {
			case 0:
				request := &api.CreateDestinationRequest{
					Name:     options.Name,
					UniqueID: chksm,
					Connection: api.DestinationConnection{
						CA:  string(caBytes),
						URL: endpoint,
					},
				}

				destination, err := client.CreateDestination(request)
				if err != nil {
					logging.S.Errorf("error creating destination: %w", err)
					return
				}

				destinationID = destination.ID
			case 1:
				request := api.UpdateDestinationRequest{
					ID:       destinations[0].ID,
					Name:     options.Name,
					UniqueID: chksm,
					Connection: api.DestinationConnection{
						CA:  string(caBytes),
						URL: endpoint,
					},
				}

				_, err := client.UpdateDestination(request)
				if err != nil {
					logging.S.Errorf("error updating destination: %w", err)
				}
			default:
				// this shouldn't happen
				logging.L.Info("unexpected result from ListDestinations")
				return
			}
		}

		grants, err := client.ListGrants(api.ListGrantsRequest{Resource: options.Name})
		if err != nil {
			logging.S.Errorf("error listing grants: %w", err)
			return
		}

		namespaces, err := k8s.Namespaces()
		if err != nil {
			logging.S.Errorf("error listing namespaces: %w", err)
			return
		}

		for _, n := range namespaces {
			g, err := client.ListGrants(api.ListGrantsRequest{Resource: fmt.Sprintf("%s.%s", options.Name, n)})
			if err != nil {
				logging.S.Errorf("error listing grants: %w", err)
				return
			}

			grants = append(grants, g...)
		}

		err = updateRoles(client, k8s, grants)
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
				Transport: &http.Transport{
					TLSClientConfig: hostTLSConfig,
				},
			},
		},
		baseURL: u.String(),
	}

	mux.Handle("/proxy/", http.StripPrefix("/proxy", jwtMiddleware(ph, cache.getJWK)))

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   handlers.CustomLoggingHandler(io.Discard, mux, logging.ZapLogFormatter),
		ErrorLog:  logging.StandardErrorLog(),
	}

	logging.L.Info("serving on port 443")

	return tlsServer.ListenAndServeTLS("", "")
}
