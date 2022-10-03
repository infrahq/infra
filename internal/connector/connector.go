package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/prometheus/client_golang/prometheus"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/metrics"
)

type Options struct {
	Server ServerOptions
	Name   string
	CACert types.StringOrFile
	CAKey  types.StringOrFile

	Addr ListenerOptions
}

type ServerOptions struct {
	URL                string
	AccessKey          types.StringOrFile
	SkipTLSVerify      bool
	TrustedCertificate types.StringOrFile
}

type ListenerOptions struct {
	HTTPS   string
	Metrics string
}

// connector stores all the dependencies for the connector operations.
type connector struct {
	k8s         *kubernetes.Kubernetes
	client      *api.Client
	destination *api.Destination
	certCache   *CertCache
	options     Options
}

func Run(ctx context.Context, options Options) error {
	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	checkSum := k8s.Checksum()
	logging.L.Debug().Str("uniqueID", checkSum).Msg("Cluster uniqueID")

	if options.Name == "" {
		autoname, err := k8s.Name(checkSum)
		if err != nil {
			logging.Errorf("k8s name error: %s", err)
			return err
		}
		options.Name = autoname
	}

	certCache := NewCertCache([]byte(options.CACert), []byte(options.CAKey))

	// Generate TLS certificates on the fly for clients
	// GenerateCertificate caches certificates
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	tlsConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return certCache.Certificate()
	}

	u, err := urlx.Parse(options.Server.URL)
	if err != nil {
		return fmt.Errorf("invalid server url: %w", err)
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
		UniqueID: checkSum,
	}

	// clone the default http transport which sets reasonable defaults
	defaultHTTPTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return errors.New("unexpected type for http.DefaultTransport")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	promRegistry := metrics.NewRegistry(internal.FullVersion())
	responseDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http_client",
		Name:      "request_duration_seconds",
		Help:      "A histogram of duration, in seconds, performing HTTP requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"host", "method", "path", "status"})
	promRegistry.MustRegister(responseDuration)

	client := &api.Client{
		Name:      "connector",
		Version:   internal.Version,
		URL:       u.String(),
		AccessKey: options.Server.AccessKey.String(),
		HTTP: http.Client{
			Transport: httpTransportFromOptions(options.Server),
		},
		Headers: http.Header{
			"Infra-Destination": {checkSum},
		},
		OnUnauthorized: func() {
			logging.Errorf("Unauthorized error; token invalid or expired. exiting.")
			cancel()
		},
		ObserveFunc: func(start time.Time, request *http.Request, response *http.Response, err error) {
			statusLabel := ""
			if response != nil {
				statusLabel = strconv.Itoa(response.StatusCode)
			}

			if err != nil {
				statusLabel = "-1"
			}

			responseDuration.With(prometheus.Labels{
				"host":   request.URL.Host,
				"method": request.Method,
				"path":   request.URL.Path,
				"status": statusLabel,
			}).Observe(time.Since(start).Seconds())
		},
	}

	con := connector{
		k8s:         k8s,
		client:      client,
		destination: destination,
		certCache:   certCache,
		options:     options,
	}
	// TODO: make polling time configurable
	repeat.Start(ctx, 30*time.Second, syncWithServer(con))

	ginutil.SetMode()
	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	proxyHost, err := urlx.Parse(k8s.Config.Host)
	if err != nil {
		return fmt.Errorf("parsing host config: %w", err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(k8s.Config.CAData); !ok {
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

	httpErrorLog := log.New(logging.NewFilteredHTTPLogger(), "", 0)
	metricsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.Metrics,
		Handler:           metrics.NewHandler(promRegistry),
		ErrorLog:          httpErrorLog,
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
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.HTTPS,
		TLSConfig:         tlsConfig,
		Handler:           router,
		ErrorLog:          httpErrorLog,
	}

	logging.Infof("starting infra connector (%s) - https:%s metrics:%s", internal.FullVersion(), tlsServer.Addr, metricsServer.Addr)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// listen for shutdown from main context.
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		defer shutdownCancel()
		err = tlsServer.Shutdown(shutdownCtx)
		logging.Warnf("shutdown: %s", err)
		wg.Done()
	}()

	err = tlsServer.ListenAndServeTLS("", "")
	wg.Wait() // must wait for shutdown to complete.
	return err
}

func httpTransportFromOptions(opts ServerOptions) *http.Transport {
	roots, err := x509.SystemCertPool()
	if err != nil {
		logging.L.Warn().Err(err).Msgf("failed to load TLS roots from system")
		roots = x509.NewCertPool()
	}

	if opts.TrustedCertificate != "" {
		if !roots.AppendCertsFromPEM([]byte(opts.TrustedCertificate)) {
			logging.Warnf("failed to load TLS CA, invalid PEM")
		}
	}

	// nolint:forcetypeassert // http.DefaultTransport is always http.Transport
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		//nolint:gosec // We may purposely set InsecureSkipVerify via a flag
		InsecureSkipVerify: opts.SkipTLSVerify,
		RootCAs:            roots,
	}
	return transport
}

func syncDestination(con connector) error {
	host, port, err := con.k8s.Endpoint()
	if err != nil {
		return fmt.Errorf("failed to lookup endpoint: %w", err)
	}

	if ipv4 := net.ParseIP(host); ipv4 == nil {
		// wait for DNS resolution if endpoint is not an IPv4 address
		if _, err := net.LookupIP(host); err != nil {
			return fmt.Errorf("host could not be resolved: %w", err)
		}
	}

	// update certificates if the host changed
	_, err = con.certCache.AddHost(host)
	if err != nil {
		return fmt.Errorf("could not update self-signed certificates")
	}

	endpoint := fmt.Sprintf("%s:%d", host, port)
	logging.Debugf("connector serving on %s", endpoint)

	namespaces, err := con.k8s.Namespaces()
	if err != nil {
		return fmt.Errorf("could not get kubernetes namespaces: %w", err)
	}

	clusterRoles, err := con.k8s.ClusterRoles()
	if err != nil {
		return fmt.Errorf("could not get kubernetes cluster-roles: %w", err)
	}

	switch {
	case con.destination.ID == 0:
		// TODO: move this warning somewhere earlier in startup
		isClusterIP, err := con.k8s.IsServiceTypeClusterIP()
		if err != nil {
			logging.Debugf("could not determine service type: %v", err)
		}

		if isClusterIP {
			logging.Warnf("registering Kubernetes connector with ClusterIP. it may not be externally accessible. if you are experiencing connectivity issues, consider switching to LoadBalancer or Ingress")
		}

		fallthrough

	case !slicesEqual(con.destination.Resources, namespaces):
		con.destination.Resources = namespaces
		fallthrough

	case !slicesEqual(con.destination.Roles, clusterRoles):
		con.destination.Roles = clusterRoles
		fallthrough

	case string(con.destination.Connection.CA) != string(con.options.CACert):
		con.destination.Connection.CA = api.PEM(con.options.CACert)
		fallthrough

	case con.destination.Connection.URL != endpoint:
		con.destination.Connection.URL = endpoint

		if err := createOrUpdateDestination(con.client, con.destination); err != nil {
			return fmt.Errorf("create or update destination: %w", err)
		}
	}
	return nil
}

func syncWithServer(con connector) func(context.Context) {
	return func(context.Context) {
		if err := syncDestination(con); err != nil {
			logging.Errorf("failed to update destination in infra: %v", err)
			return
		}

		grants, err := con.client.ListGrants(api.ListGrantsRequest{Resource: con.destination.Name})
		if err != nil {
			logging.Errorf("error listing grants: %v", err)
			return
		}

		namespaces, err := con.k8s.Namespaces()
		if err != nil {
			logging.Errorf("could not get kubernetes namespaces: %v", err)
			return
		}

		// TODO(https://github.com/infrahq/infra/issues/2422): support wildcard resource searches
		for _, n := range namespaces {
			g, err := con.client.ListGrants(api.ListGrantsRequest{Resource: fmt.Sprintf("%s.%s", con.destination.Name, n)})
			if err != nil {
				logging.Errorf("error listing grants: %v", err)
				return
			}

			grants.Items = append(grants.Items, g.Items...)
		}

		err = updateRoles(con.client, con.k8s, grants.Items)
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
		Version:    internal.FullVersion(),
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
		Version:    internal.FullVersion(),
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
