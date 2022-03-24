package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/goware/urlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/metrics"
	"github.com/infrahq/infra/secrets"
	"github.com/infrahq/infra/uid"
)

type Options struct {
	Server        string `mapstructure:"server"`
	Name          string `mapstructure:"name"`
	AccessKey     string `mapstructure:"accessKey"`
	TLSCache      string `mapstructure:"tlsCache"`
	TLSCert       string `mapstructure:"tlsCert"`
	TLSKey        string `mapstructure:"tlsKey"`
	SkipTLSVerify bool   `mapstructure:"skipTLSVerify"`
}

type jwkCache struct {
	mu          sync.Mutex
	key         *jose.JSONWebKey
	lastChecked time.Time

	client  *http.Client
	baseURL string
}

type localDetails struct {
	endpoint      string
	ca            string
	name          string
	chksm         string
	destinationID uid.ID
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

type getJWKFunc func() (*jose.JSONWebKey, error)

func jwtMiddleware(getJWK getJWKFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")

		raw := strings.ReplaceAll(authorization, "Bearer ", "")
		if raw == "" {
			logging.L.Debug("no bearer token found")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		tok, err := jwt.ParseSigned(raw)
		if err != nil {
			logging.L.Debug("invalid jwt signature")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		key, err := getJWK()
		if err != nil {
			logging.L.Debug("could not get jwk")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		var claims struct {
			jwt.Claims
			claims.Custom
		}

		out := make(map[string]interface{})
		if err := tok.Claims(key, &claims, &out); err != nil {
			logging.L.Debug("invalid token claims")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		err = claims.Claims.Validate(jwt.Expected{
			Time: time.Now(),
		})

		switch {
		case errors.Is(err, jwt.ErrExpired):
			logging.S.Debugf("expired JWT %s", err.Error())
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		case err != nil:
			logging.S.Debugf("invalid JWT %s", err.Error())
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if err := validator.New().Struct(claims.Custom); err != nil {
			logging.L.Debug("JWT custom claims not valid")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.Set("email", claims.Email)
		c.Set("machine", claims.Machine)
		c.Set("groups", claims.Groups)
		c.Set("provider", claims.Provider)

		c.Next()
	}
}

func proxyMiddleware(proxy *httputil.ReverseProxy, bearerToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, ok := c.MustGet("email").(string)
		if !ok {
			logging.S.Debug("required field 'email' not found")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		machine, ok := c.MustGet("machine").(string)
		if !ok {
			logging.S.Debug("required field 'machine' not found")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		groups, ok := c.MustGet("groups").([]string)
		if !ok {
			logging.S.Debug("required field 'groups' not found")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		provider, ok := c.MustGet("provider").(string)
		if !ok {
			logging.S.Debug("required field 'provider' not found")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		userParts := []string{email}
		if len(provider) > 0 {
			userParts = append([]string{provider}, userParts...)
		}

		switch {
		case email != "":
			c.Request.Header.Set("Impersonate-User", strings.Join(userParts, ":"))
		case machine != "":
			c.Request.Header.Set("Impersonate-User", fmt.Sprintf("machine:%s", machine))
		default:
			logging.S.Debug("unable to determine identity")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		for _, g := range groups {
			c.Request.Header.Add("Impersonate-Group", g)
		}

		c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// UpdateRoles converts infra grants to role-bindings in the current cluster
func updateRoles(c *api.Client, k *kubernetes.Kubernetes, grants []api.Grant) error {
	logging.L.Debug("syncing local grants from infra configuration")

	crSubjects := make(map[string][]rbacv1.Subject)                           // cluster-role: subject
	crnSubjects := make(map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) // cluster-role+namespace: subject

	for _, g := range grants {
		var name, kind string

		id, err := g.Subject.ID()
		if err != nil {
			return err
		}

		switch {
		case g.Subject.IsGroup():
			group, err := c.GetGroup(id)
			if err != nil {
				return err
			}

			name = group.Name
			kind = rbacv1.GroupKind

		case g.Subject.IsUser():
			user, err := c.GetUser(id)
			if err != nil {
				return err
			}

			name = user.Email
			kind = rbacv1.UserKind

			provider, err := c.GetProvider(user.ProviderID)
			if err != nil {
				return err
			}

			name = provider.Name + ":" + name

		case g.Subject.IsMachine():
			machine, err := c.GetMachine(id)
			if err != nil {
				return err
			}

			name = fmt.Sprintf("machine:%s", machine.Name)
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

	serverName := "infra-connector"

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	var tlsConfig *tls.Config

	if options.TLSCert != "" || options.TLSKey != "" {
		certBytes, err := ioutil.ReadFile(options.TLSCert)
		if err != nil {
			return err
		}

		keypair, err := tls.LoadX509KeyPair(options.TLSCert, options.TLSKey)
		if err != nil {
			return err
		}

		tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{keypair},
		}

		if err := manager.Cache.Put(context.TODO(), serverName, certBytes); err != nil {
			return err
		}
	} else {
		tlsConfig := manager.TLSConfig()
		tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, serverName)
	}

	basicSecretStorage := map[string]secrets.SecretStorage{
		"env":       secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{}),
		"file":      secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{}),
		"plaintext": secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{}),
	}

	u, err := urlx.Parse(options.Server)
	if err != nil {
		logging.S.Errorf("server: %w", err)
	}

	// server is localhost which should never be the case. try to infer the actual host
	if strings.HasPrefix(u.Host, "localhost") {
		server, err := k8s.Service("server")
		if err != nil {
			logging.S.Warnf("no cluster-local infra server found for %q. check connector configurations", u.Host)
		} else {
			host := fmt.Sprintf("%s.%s", server.ObjectMeta.Name, server.ObjectMeta.Namespace)
			logging.S.Debugf("using cluster-local infra server at %q instead of %q", host, u.Host)
			u.Host = host
		}
	}

	u.Scheme = "https"

	localDetails := &localDetails{
		name:  options.Name,
		chksm: chksm,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repeat.Start(ctx, 5*time.Second, func(context.Context) {
		accessKey, err := secrets.GetSecret(options.AccessKey, basicSecretStorage)
		if err != nil {
			logging.S.Infof("%w", err)
			return
		}

		client := &api.Client{
			URL:       u.String(),
			AccessKey: accessKey,
			HTTP: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: hostTLSConfig,
				},
			},
		}

		caBytes, err := manager.Cache.Get(context.TODO(), serverName)
		if err != nil {
			if errors.Is(err, autocert.ErrCacheMiss) {
				logging.S.Debugf("failed loading CA: %s", err.Error())
				return
			} else {
				logging.S.Errorf("cache get: %s", err.Error())
				return
			}
		}

		host, port, err := k8s.Endpoint()
		if err != nil {
			logging.S.Errorf("endpoint: %w", err)
			return
		}

		if ipv4 := net.ParseIP(host); ipv4 == nil {
			// wait for DNS resolution if endpoint is not an IPv4 address
			if _, err := net.LookupIP(host); err != nil {
				logging.L.Error("host could not be resolved")
				return
			}
		}

		endpoint := fmt.Sprintf("%s:%d", host, port)
		logging.S.Debugf("connector serving on %s", endpoint)

		if localDetails.destinationID == 0 {
			localDetails.ca = string(caBytes)
			localDetails.endpoint = endpoint

			isClusterIP, err := k8s.IsServiceTypeClusterIP()
			if err != nil {
				logging.S.Debugf("could not check destination service type: %w", err)
			}

			if isClusterIP {
				logging.S.Warn("registering with cluster IP, it may not be externally accessible without an ingress or load balancer")
			}

			err = registerDestination(client, localDetails)
			if err != nil {
				logging.S.Errorf("initializing destination: %w", err)
				return
			}
		} else if localDetails.endpoint != endpoint || localDetails.ca != string(caBytes) {
			localDetails.ca = string(caBytes)
			localDetails.endpoint = endpoint

			err = refreshDestination(client, localDetails)
			if err != nil {
				logging.S.Errorf("initializing destination: %w", err)
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

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

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

	proxyHost, err := urlx.Parse(k8s.Config.Host)
	if err != nil {
		return fmt.Errorf("parsing host config: %w", err)
	}

	caCert, err := kubernetes.CA()
	if err != nil {
		return fmt.Errorf("reading CA file: %w", err)
	}

	certPool := x509.NewCertPool()

	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return errors.New("could not append CA to client cert bundle")
	}

	proxy := httputil.NewSingleHostReverseProxy(proxyHost)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		},
	}

	router.Use(
		metrics.Middleware(),
		jwtMiddleware(cache.getJWK),
		proxyMiddleware(proxy, k8s.Config.BearerToken),
	)

	metrics := gin.New()
	metrics.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	metricsServer := &http.Server{
		Addr:     ":9090",
		Handler:  metrics,
		ErrorLog: logging.StandardErrorLog(),
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			logging.S.Errorf("server: %w", err)
		}
	}()

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   router,
		ErrorLog:  logging.StandardErrorLog(),
	}

	logging.S.Infof("starting infra (%s) - https:%s metrics:%s", internal.Version, tlsServer.Addr, metricsServer.Addr)

	return tlsServer.ListenAndServeTLS("", "")
}

// registerDestination creates a destination in the infra server if it does not exist
func registerDestination(client *api.Client, local *localDetails) error {
	destinations, err := client.ListDestinations(api.ListDestinationsRequest{Name: "", UniqueID: local.chksm})
	if err != nil {
		return fmt.Errorf("error listing destinations: %w", err)
	}

	if len(destinations) == 0 {
		request := &api.CreateDestinationRequest{
			Name:     local.name,
			UniqueID: local.chksm,
			Connection: api.DestinationConnection{
				CA:  local.ca,
				URL: local.endpoint,
			},
		}

		destination, err := client.CreateDestination(request)
		if err != nil {
			return fmt.Errorf("error creating destination: %w", err)
		}

		local.destinationID = destination.ID
	} else {
		local.destinationID = destinations[0].ID
		return refreshDestination(client, local)
	}

	return nil
}

func refreshDestination(client *api.Client, local *localDetails) error {
	logging.S.Debug("updating information at server")

	request := api.UpdateDestinationRequest{
		ID:       local.destinationID,
		Name:     local.name,
		UniqueID: local.chksm,
		Connection: api.DestinationConnection{
			CA:  local.ca,
			URL: local.endpoint,
		},
	}

	if _, err := client.UpdateDestination(request); err != nil {
		return fmt.Errorf("error updating existing destination: %w", err)
	}

	return nil
}
