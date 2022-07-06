package connector

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/infrahq/secrets"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
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
		Headers: http.Header{
			"Infra-Destination": {chksm},
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
