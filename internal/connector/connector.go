package connector

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/infrahq/secrets"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/metrics"
)

type Options struct {
	Server        string
	Name          string
	AccessKey     string
	CACert        string
	CAKey         string
	SkipTLSVerify bool
}

type authenticator struct {
	mu          sync.Mutex
	key         *jose.JSONWebKey
	lastChecked time.Time

	client  httpClient
	baseURL string
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (j *authenticator) getJWK() (*jose.JSONWebKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.lastChecked.IsZero() && time.Now().Before(j.lastChecked.Add(JWKCacheRefresh)) {
		return j.key, nil
	}

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, fmt.Sprintf("%s/.well-known/jwks.json", j.baseURL), nil)
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

	j.lastChecked = time.Now().UTC()
	j.key = &response.Keys[0]

	return &response.Keys[0], nil
}

func newAuthenticator(url string, options Options) *authenticator {
	// nolint:forcetypeassert // http.DefaultTransport is always http.Transport
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		//nolint:gosec // We may purposely set InsecureSkipVerify via a flag
		InsecureSkipVerify: options.SkipTLSVerify,
	}

	return &authenticator{
		client:  &http.Client{Transport: transport},
		baseURL: url,
	}
}

var JWKCacheRefresh = 5 * time.Minute

func (j *authenticator) Authenticate(req *http.Request) (claims.Custom, error) {
	c := claims.Custom{}
	authHeader := req.Header.Get("Authorization")

	raw := strings.TrimPrefix(authHeader, "Bearer ")
	if raw == "" {
		return c, fmt.Errorf("no bearer token found")
	}

	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		return c, fmt.Errorf("invalid JWT signature: %w", err)
	}

	key, err := j.getJWK()
	if err != nil {
		return c, fmt.Errorf("get JWK from server: %w", err)
	}

	var allClaims struct {
		jwt.Claims
		claims.Custom
	}
	if err := tok.Claims(key, &allClaims); err != nil {
		return c, fmt.Errorf("invalid token claims: %w", err)
	}

	err = allClaims.Claims.Validate(jwt.Expected{Time: time.Now().UTC()})
	switch {
	case errors.Is(err, jwt.ErrExpired):
		return c, err
	case err != nil:
		return c, fmt.Errorf("invalid JWT %w", err)
	}

	if allClaims.Custom.Name == "" {
		return c, fmt.Errorf("no username in JWT claims")
	}

	return allClaims.Custom, nil
}

func proxyMiddleware(
	proxy *httputil.ReverseProxy,
	authn *authenticator,
	bearerToken string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		claim, err := authn.Authenticate(c.Request)
		if err != nil {
			logging.L.Info().Err(err).Msgf("failed to authenticate request")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Request.Header.Set("Impersonate-User", claim.Name)
		for _, g := range claim.Groups {
			c.Request.Header.Add("Impersonate-Group", g)
		}

		c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// UpdateRoles converts infra grants to role-bindings in the current cluster
func updateRoles(c *api.Client, k *kubernetes.Kubernetes, grants []api.Grant) error {
	logging.Debugf("syncing local grants from infra configuration")

	crSubjects := make(map[string][]rbacv1.Subject)                           // cluster-role: subject
	crnSubjects := make(map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) // cluster-role+namespace: subject

	for _, g := range grants {
		var name, kind string

		if g.Privilege == "connect" {
			continue
		}

		switch {
		case g.Group != 0:
			group, err := c.GetGroup(g.Group)
			if err != nil {
				return err
			}

			name = group.Name
			kind = rbacv1.GroupKind
		case g.User != 0:
			user, err := c.GetUser(g.User)
			if err != nil {
				return err
			}

			name = user.Name
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
		// <cluster>
		case 1:
			crn.ClusterRole = g.Privilege
			crSubjects[g.Privilege] = append(crSubjects[g.Privilege], subj)

		// <cluster>.<namespace>
		case 2:
			crn.ClusterRole = g.Privilege
			crn.Namespace = parts[1]
			crnSubjects[crn] = append(crnSubjects[crn], subj)

		default:
			logging.Warnf("invalid grant resource: %s", g.Resource)
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

type CertCache struct {
	mu     sync.Mutex
	caCert []byte
	caKey  []byte
	hosts  []string
	cert   *tls.Certificate
}

func (c *CertCache) AddHost(host string) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, h := range c.hosts {
		if h == host {
			return c.cert, nil
		}
	}

	c.hosts = append(c.hosts, host)

	logging.Debugf("generating certificate for: %v", c.hosts)

	ca, err := tls.X509KeyPair(c.caCert, c.caKey)
	if err != nil {
		return nil, err
	}

	caCert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	certPEM, keyPEM, err := certs.GenerateCertificate(c.hosts, caCert, ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	c.cert = &tlsCert

	return c.cert, nil
}

// readCertificate is a threadsafe way to read the certificate
func (c *CertCache) readCertificate() *tls.Certificate {
	c.mu.Lock()
	cert := c.cert
	c.mu.Unlock()
	return cert
}

// Certificate returns a TLS certificate for the connector, if one does not exist it is generated for the empty host
func (c *CertCache) Certificate() (*tls.Certificate, error) {
	cert := c.readCertificate()
	if cert == nil {
		// the host is not available externally, or this would have been set
		// set to an empty host for the liveness check to resolve from the same host
		return c.AddHost("")
	}

	return cert, nil
}

func NewCertCache(caCertPEM []byte, caKeyPem []byte) *CertCache {
	return &CertCache{caCert: caCertPEM, caKey: caKeyPem}
}

func Run(ctx context.Context, options Options) error {
	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	chksm, err := k8s.Checksum()
	if err != nil {
		logging.Errorf("k8s checksum error: %s", err)
		return err
	}

	if options.Name == "" {
		autoname, err := k8s.Name(chksm)
		if err != nil {
			logging.Errorf("k8s name error: %s", err)
			return err
		}
		options.Name = autoname
	}

	caCertPEM, err := os.ReadFile(options.CACert)
	if err != nil {
		return err
	}

	caKeyPEM, err := os.ReadFile(options.CAKey)
	if err != nil {
		return err
	}

	certCache := NewCertCache(caCertPEM, caKeyPEM)

	// Generate TLS certificates on the fly for clients
	// GenerateCertificate caches certificates
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	tlsConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return certCache.Certificate()
	}

	basicSecretStorage := map[string]secrets.SecretStorage{
		"env":       secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{}),
		"file":      secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{}),
		"plaintext": secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{}),
	}

	u, err := urlx.Parse(options.Server)
	if err != nil {
		logging.Errorf("server: %s", err)
	}

	// server is localhost which should never be the case. try to infer the actual host
	if strings.HasPrefix(u.Host, "localhost") {
		server, err := k8s.Service("server")
		if err != nil {
			logging.Warnf("no cluster-local infra server found for %q. check connector configurations", u.Host)
		} else {
			host := fmt.Sprintf("%s.%s", server.ObjectMeta.Name, server.ObjectMeta.Namespace)
			logging.Debugf("using cluster-local infra server at %q instead of %q", host, u.Host)
			u.Host = host
		}
	}

	u.Scheme = "https"

	destination := &api.Destination{
		Name:     options.Name,
		UniqueID: chksm,
	}

	// clone the default http transport which sets reasonable defaults
	defaultHTTPTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return errors.New("unexpected type for http.DefaultTransport")
	}

	transport := defaultHTTPTransport.Clone()
	transport.TLSClientConfig = &tls.Config{
		//nolint:gosec // We may purposely set InsecureSkipVerify via a flag
		InsecureSkipVerify: options.SkipTLSVerify,
	}

	accessKey, err := secrets.GetSecret(options.AccessKey, basicSecretStorage)
	if err != nil {
		return err
	}

	client := &api.Client{
		Name:      "connector",
		Version:   internal.Version,
		URL:       u.String(),
		AccessKey: accessKey,
		HTTP: http.Client{
			Transport: transport,
		},
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	repeat.Start(ctx, 5*time.Second, syncWithServer(k8s, client, destination, certCache, caCertPEM))

	ginutil.SetMode()
	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	proxyHost, err := urlx.Parse(k8s.Config.Host)
	if err != nil {
		return fmt.Errorf("parsing host config: %w", err)
	}

	clusterCACert, err := kubernetes.CA()
	if err != nil {
		return fmt.Errorf("reading CA file: %w", err)
	}

	certPool := x509.NewCertPool()

	if ok := certPool.AppendCertsFromPEM(clusterCACert); !ok {
		return errors.New("could not append CA to client cert bundle")
	}

	proxyTransport := defaultHTTPTransport.Clone()
	proxyTransport.ForceAttemptHTTP2 = false
	proxyTransport.TLSClientConfig = &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	proxy := httputil.NewSingleHostReverseProxy(proxyHost)
	proxy.Transport = proxyTransport

	promRegistry := setupMetrics()
	httpErrorLog := log.New(logging.NewFilteredHTTPLogger(), "", 0)
	metricsServer := &http.Server{
		Addr:     ":9090",
		Handler:  metrics.NewHandler(promRegistry),
		ErrorLog: httpErrorLog,
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			logging.Errorf("server: %s", err)
		}
	}()

	authn := newAuthenticator(u.String(), options)
	router.Use(
		metrics.Middleware(promRegistry),
		proxyMiddleware(proxy, authn, k8s.Config.BearerToken),
	)
	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   router,
		ErrorLog:  httpErrorLog,
	}

	logging.Infof("starting infra connector (%s) - https:%s metrics:%s", internal.FullVersion(), tlsServer.Addr, metricsServer.Addr)

	return tlsServer.ListenAndServeTLS("", "")
}

func syncWithServer(k8s *kubernetes.Kubernetes, client *api.Client, destination *api.Destination, certCache *CertCache, caCertPEM []byte) func(context.Context) {

	return func(context.Context) {
		host, port, err := k8s.Endpoint()
		if err != nil {
			logging.Errorf("failed to lookup endpoint: %v", err)
			return
		}

		if ipv4 := net.ParseIP(host); ipv4 == nil {
			// wait for DNS resolution if endpoint is not an IPv4 address
			if _, err := net.LookupIP(host); err != nil {
				logging.Errorf("host could not be resolved")
				return
			}
		}

		// update certificates if the host changed
		_, err = certCache.AddHost(host)
		if err != nil {
			logging.Errorf("could not update self-signed certificates")
			return
		}

		endpoint := fmt.Sprintf("%s:%d", host, port)
		logging.Debugf("connector serving on %s", endpoint)

		namespaces, err := k8s.Namespaces()
		if err != nil {
			logging.Errorf("could not get kubernetes namespaces: %v", err)
			return
		}

		clusterRoles, err := k8s.ClusterRoles()
		if err != nil {
			logging.Errorf("could not get kubernetes cluster-roles: %v", err)
			return
		}

		switch {
		case destination.ID == 0:
			isClusterIP, err := k8s.IsServiceTypeClusterIP()
			if err != nil {
				logging.Debugf("could not determine service type: %v", err)
			}

			if isClusterIP {
				logging.Warnf("registering Kubernetes connector with ClusterIP. it may not be externally accessible. if you are experiencing connectivity issues, consider switching to LoadBalancer or Ingress")
			}

			fallthrough

		case !slicesEqual(destination.Resources, namespaces):
			destination.Resources = namespaces
			fallthrough

		case !slicesEqual(destination.Roles, clusterRoles):
			destination.Roles = clusterRoles
			fallthrough

		case !bytes.Equal([]byte(destination.Connection.CA), caCertPEM):
			destination.Connection.CA = api.PEM(caCertPEM)
			fallthrough

		case destination.Connection.URL != endpoint:
			destination.Connection.URL = endpoint

			if err := createOrUpdateDestination(client, destination); err != nil {
				logging.Errorf("initializing destination: %v", err)
				return
			}
		}

		grants, err := client.ListGrants(api.ListGrantsRequest{Resource: destination.Name})
		if err != nil {
			logging.Errorf("error listing grants: %v", err)
			return
		}

		// TODO(https://github.com/infrahq/infra/issues/2422): support wildcard resource searches
		for _, n := range namespaces {
			g, err := client.ListGrants(api.ListGrantsRequest{Resource: fmt.Sprintf("%s.%s", destination.Name, n)})
			if err != nil {
				logging.Errorf("error listing grants: %v", err)
				return
			}

			grants.Items = append(grants.Items, g.Items...)
		}

		err = updateRoles(client, k8s, grants.Items)
		if err != nil {
			logging.Errorf("error updating grants: %v", err)
			return
		}
	}
}

// createOrUpdateDestination creates a destination in the infra server if it does not exist and updates it if it does
func createOrUpdateDestination(client *api.Client, local *api.Destination) error {
	if local.ID != 0 {
		return updateDestination(client, local)
	}

	destinations, err := client.ListDestinations(api.ListDestinationsRequest{UniqueID: local.UniqueID})
	if err != nil {
		return fmt.Errorf("error listing destinations: %w", err)
	}

	if destinations.Count > 0 {
		local.ID = destinations.Items[0].ID
		return updateDestination(client, local)
	}

	request := &api.CreateDestinationRequest{
		Name:       local.Name,
		UniqueID:   local.UniqueID,
		Connection: local.Connection,
		Resources:  local.Resources,
		Roles:      local.Roles,
	}

	destination, err := client.CreateDestination(request)
	if err != nil {
		return fmt.Errorf("error creating destination: %w", err)
	}

	local.ID = destination.ID
	return nil
}

// updateDestination updates a destination in the infra server
func updateDestination(client *api.Client, local *api.Destination) error {
	logging.Debugf("updating information at server")

	request := api.UpdateDestinationRequest{
		ID:         local.ID,
		Name:       local.Name,
		UniqueID:   local.UniqueID,
		Connection: local.Connection,
		Resources:  local.Resources,
		Roles:      local.Roles,
	}

	if _, err := client.UpdateDestination(request); err != nil {
		return fmt.Errorf("error updating existing destination: %w", err)
	}

	return nil
}

// slicesEqual checks if two sorted slices of strings are equal
func slicesEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}
