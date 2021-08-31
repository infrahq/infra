package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/timer"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Options struct {
	Registry       string
	Name           string
	Endpoint       string
	ForceTLSVerify bool
	APIKey         string
}

type RegistrationInfo struct {
	CA              string
	ClusterEndpoint string
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

	res, err := j.client.Get(j.baseURL + "/.well-known/jwks.json")
	if err != nil {
		return nil, err
	}

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

func jwtMiddleware(getjwk GetJWKFunc, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("X-Infra-Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)
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
		cl := jwt.Claims{}
		if err := tok.Claims(key, &cl, &out); err != nil {
			logging.L.Debug("Invalid token claims")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		err = cl.Validate(jwt.Expected{
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

		email, ok := out["email"].(string)
		if !ok {
			logging.L.Debug("Email not found in claims")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, HttpContextKeyEmail{}, email)
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
			RootCAs: caCertPool,
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Sometimes the kubernetes proxy strips query string for Upgrade requests
		// so we need to put that in the request body
		if r.Header.Get("X-Infra-Query") != "" {
			r.URL.RawQuery = r.Header.Get("X-Infra-Query")
		}

		email, ok := r.Context().Value(HttpContextKeyEmail{}).(string)
		if !ok {
			logging.L.Debug("Proxy handler unable to retrieve email from context")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Header.Del("X-Infra-Authorization")
		r.Header.Set("Impersonate-User", email)
		r.Header.Add("Authorization", "Bearer "+bearerToken)

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
	tlsConfig := &tls.Config{}
	if !options.ForceTLSVerify {
		// TODO (https://github.com/infrahq/infra/issues/174)
		// Find a way to re-use the built-in TLS verification code vs
		// this custom code based on the official go TLS example code
		// which states this is approximately the same.
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
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

	ctx := context.WithValue(context.Background(), api.ContextServerVariables, map[string]string{"basePath": "v1"})
	ctx = context.WithValue(ctx, api.ContextAccessToken, options.APIKey)
	config := api.NewConfiguration()
	config.Host = options.Registry
	config.Scheme = "https"
	config.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	client := api.NewAPIClient(config)

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	timer := timer.Timer{}
	timer.Start(5, func() {
		ca, err := k8s.CA()
		if err != nil {
			logging.L.Error(err.Error())
			return
		}

		endpoint := options.Endpoint
		if endpoint == "" {
			endpoint, err = k8s.Endpoint()
			if err != nil {
				logging.L.Error(err.Error())
				return
			}
		}

		namespace, err := k8s.Namespace()
		if err != nil {
			logging.L.Error(err.Error())
			return
		}

		name := options.Name
		if name == "" {
			name, err = k8s.Name()
			if err != nil {
				logging.L.Error(err.Error())
				return
			}
		}

		saToken, err := k8s.SaToken()
		if err != nil {
			logging.L.Error(err.Error())
			return
		}

		destination, _, err := client.DestinationsApi.CreateDestination(ctx).Body(api.DestinationCreateRequest{
			Name: name,
			Kubernetes: &api.DestinationKubernetes{
				Ca:        string(ca),
				Endpoint:  endpoint,
				Namespace: namespace,
				SaToken:   saToken,
			},
		}).Execute()
		if err != nil {
			logging.L.Error(err.Error())
			return
		}

		roles, _, err := client.RolesApi.ListRoles(ctx).DestinationId(destination.Id).Execute()
		if err != nil {
			logging.L.Error(err.Error())
		}

		// convert the response into an easy to use role-user form
		var rbs []kubernetes.RoleBinding
		for _, r := range roles {
			var users []string
			for _, u := range r.Users {
				users = append(users, u.Email)
			}
			rbs = append(rbs, kubernetes.RoleBinding{Role: r.Name, Users: users})
		}

		err = k8s.UpdateRoles(rbs)
		if err != nil {
			logging.L.Error(err.Error())
			return
		}
	})
	defer timer.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	remote, err := url.Parse(k8s.Config.Host)
	if err != nil {
		return err
	}

	ca, err := ioutil.ReadFile(k8s.Config.TLSClientConfig.CAFile)
	if err != nil {
		return err
	}

	ph, err := proxyHandler(ca, k8s.Config.BearerToken, remote)
	if err != nil {
		return err
	}

	u, err := urlx.Parse(options.Registry)
	if err != nil {
		return err
	}
	u.Scheme = "https"

	cache := jwkCache{
		client: &http.Client{
			Transport: &BearerTransport{
				Token: options.APIKey,
				Transport: &http.Transport{
					TLSClientConfig: tlsConfig,
				},
			},
		},
		baseURL: u.String(),
	}

	mux.Handle("/proxy/", jwtMiddleware(cache.getjwk, ph))

	logging.L.Info("serving on port 80")
	return http.ListenAndServe(":80", handlers.LoggingHandler(os.Stdout, mux))
}
