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

	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/timer"
	v1 "github.com/infrahq/infra/internal/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcMetadata "google.golang.org/grpc/metadata"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Options struct {
	Registry      string
	Name          string
	Endpoint      string
	SkipTLSVerify bool
	APIKey        string
}

type RoleBinding struct {
	Role  string
	Users []string
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

func withClientAuthUnaryInterceptor(token string) grpc.DialOption {
	return grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(grpcMetadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	})
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
	u, err := urlx.Parse(options.Registry)
	if err != nil {
		return err
	}

	registry := u.Host
	if u.Port() == "" {
		registry += ":443"
	}

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: options.SkipTLSVerify})
	conn, err := grpc.Dial(registry, grpc.WithTransportCredentials(creds), withClientAuthUnaryInterceptor(options.APIKey))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := v1.NewV1Client(conn)

	kubernetes, err := NewKubernetes()
	if err != nil {
		return err
	}

	uri, err := urlx.Parse(options.Registry)
	if err != nil {
		return err
	}

	uri.Scheme = "https"

	timer := timer.Timer{}
	timer.Start(5, func() {
		ca, err := kubernetes.CA()
		if err != nil {
			fmt.Println(err)
			return
		}

		endpoint := options.Endpoint
		if endpoint == "" {
			endpoint, err = kubernetes.Endpoint()
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		namespace, err := kubernetes.Namespace()
		if err != nil {
			fmt.Println(err)
			return
		}

		name := options.Name
		if name == "" {
			name, err = kubernetes.Name()
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		saToken, err := kubernetes.SaToken()
		if err != nil {
			fmt.Println(err)
			return
		}

		res, err := client.CreateDestination(context.Background(), &v1.CreateDestinationRequest{
			Name: name,
			Type: v1.DestinationType_KUBERNETES,
			Kubernetes: &v1.CreateDestinationRequest_Kubernetes{
				Ca:        string(ca),
				Endpoint:  endpoint,
				Namespace: namespace,
				SaToken:   saToken,
			},
		})
		if err != nil {
			fmt.Println(err)
			return
		}

		rolesRes, err := client.ListRoles(context.Background(), &v1.ListRolesRequest{
			DestinationId: res.Id,
		})
		if err != nil {
			fmt.Println(err)
		}

		// convert the response into an easy to use role-user form
		var rbs []RoleBinding
		for _, r := range rolesRes.Roles {
			var users []string
			for _, u := range r.Users {
				users = append(users, u.Email)
			}
			rbs = append(rbs, RoleBinding{Role: r.Name, Users: users})
		}

		err = kubernetes.UpdateRoles(rbs)
		if err != nil {
			fmt.Println(err)
			return
		}
	})
	defer timer.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	remote, err := url.Parse(kubernetes.config.Host)
	if err != nil {
		return err
	}

	ca, err := ioutil.ReadFile(kubernetes.config.TLSClientConfig.CAFile)
	if err != nil {
		return err
	}

	ph, err := proxyHandler(ca, kubernetes.config.BearerToken, remote)
	if err != nil {
		return err
	}

	u.Scheme = "https"

	cache := jwkCache{
		client: &http.Client{
			Transport: &BearerTransport{
				Token: options.APIKey,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: options.SkipTLSVerify,
					},
				},
			},
		},
		baseURL: u.String(),
	}

	mux.Handle("/proxy/", jwtMiddleware(cache.getjwk, ph))

	fmt.Println("serving on port 80")
	return http.ListenAndServe(":80", handlers.LoggingHandler(os.Stdout, mux))
}
