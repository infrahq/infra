package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/goware/urlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/metrics"
	"github.com/infrahq/infra/uid"
)

func Run(ctx context.Context, options Options) error {
	switch options.Kind {
	case "kubernetes":
		return runKubernetesConnector(ctx, options)
	case "ssh":
		return runSSHConnector(ctx, options)
	default:
		return fmt.Errorf("unsupported connector kind: %v", options.Kind)
	}
}

type Options struct {
	Server ServerOptions
	Addr   ListenerOptions
	Name   string // TODO: make this required
	Kind   string

	// EndpointAddr is the host:port address that clients should use to connect
	// to this destination.
	// If this value is empty then the host:port will be looked up.
	// For kubernetes the lookup is done using kube API, using the name of the connector
	// service.
	// For SSH the lookup is done from the network interface addresses.
	// This value is sent to the infra API server to update the
	// Destination.Connection.URL.
	EndpointAddr types.HostPort

	SSH SSHOptions

	// Kubernetes specific options below here
	CACert types.StringOrFile
	CAKey  types.StringOrFile
}

type ServerOptions struct {
	URL                types.URL
	AccessKey          types.StringOrFile
	SkipTLSVerify      bool
	TrustedCertificate types.StringOrFile
}

func (o Options) APIClient() *api.Client {
	o.Server.URL.Scheme = "https"
	return &api.Client{
		Name:      "connector",
		Version:   internal.Version,
		URL:       o.Server.URL.String(),
		AccessKey: o.Server.AccessKey.String(),
		HTTP: http.Client{
			Transport: httpTransportFromOptions(o.Server),
		},
		Headers: http.Header{
			"Infra-Destination-Name": {o.Name},
		},
	}
}

type ListenerOptions struct {
	HTTP    string
	HTTPS   string
	Metrics string
}

// connector stores all the dependencies for the connector operations.
type connector struct {
	k8s         kubeClient
	client      apiClient
	destination *api.Destination
	certCache   *CertCache
	options     Options
}

type apiClient interface {
	ListGrants(ctx context.Context, req api.ListGrantsRequest) (*api.ListResponse[api.Grant], error)
	ListDestinationAccess(ctx context.Context, req api.ListDestinationAccessRequest) (*api.ListDestinationAccessResponse, error)
	ListDestinations(ctx context.Context, req api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error)
	CreateDestination(ctx context.Context, req *api.CreateDestinationRequest) (*api.Destination, error)
	UpdateDestination(ctx context.Context, req api.UpdateDestinationRequest) (*api.Destination, error)

	// GetGroup and GetUser are used to retrieve the name of the group or user.
	// TODO: we can remove these calls to GetGroup and GetUser by including
	// the name of the group or user in the ListGrants response.
	GetGroup(ctx context.Context, id uid.ID) (*api.Group, error)
	GetUser(ctx context.Context, id uid.ID) (*api.User, error)
}

type kubeClient interface {
	Namespaces() ([]string, error)
	ClusterRoles() ([]string, error)
	IsServiceTypeClusterIP() (bool, error)
	Endpoint() (string, int, error)

	UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error
	UpdateRoleBindings(subjects map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) error
}

func runKubernetesConnector(ctx context.Context, options Options) error {
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

	options.Server.URL.Scheme = "https"
	destination := &api.Destination{
		Name:     options.Name,
		Kind:     "kubernetes",
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
	promRegistry.MustRegister(responseDuration, metrics.RequestDuration)

	client := options.APIClient()
	client.OnUnauthorized = func() {
		logging.Errorf("Unauthorized error; token invalid or expired. exiting.")
		cancel()
	}
	client.ObserveFunc = func(start time.Time, request *http.Request, response *http.Response, err error) {
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
	}

	group, ctx := errgroup.WithContext(ctx)

	con := connector{
		k8s:         k8s,
		client:      client,
		destination: destination,
		certCache:   certCache,
		options:     options,
	}
	group.Go(func() error {
		backOff := &backoff.ExponentialBackOff{
			InitialInterval:     2 * time.Second,
			MaxInterval:         time.Minute,
			RandomizationFactor: 0.2,
			Multiplier:          1.5,
		}
		waiter := repeat.NewWaiter(backOff)
		fn := func(ctx context.Context, grants []api.Grant) error {
			return updateRoles(ctx, con.client, con.k8s, grants)
		}
		return syncGrantsToDestination(ctx, con, waiter, fn)
	})
	group.Go(func() error {
		// TODO: how long should this wait? Use exponential backoff on error?
		waiter := repeat.NewWaiter(backoff.NewConstantBackOff(30 * time.Second))
		for {
			if err := syncDestination(ctx, con); err != nil {
				logging.Errorf("failed to update destination in infra: %v", err)
			} else {
				waiter.Reset()
			}
			if err := waiter.Wait(ctx); err != nil {
				return err
			}
		}
	})

	router := http.NewServeMux()
	router.HandleFunc("/healthz", healthHandler)

	kubeAPIAddr, err := urlx.Parse(k8s.Config.Host)
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

	httpErrorLog := logging.HTTPErrorLog(zerolog.WarnLevel)

	proxy := httputil.NewSingleHostReverseProxy(kubeAPIAddr)
	proxy.Transport = proxyTransport
	proxy.ErrorLog = httpErrorLog

	metricsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.Metrics,
		Handler:           metrics.NewHandler(promRegistry),
		ErrorLog:          httpErrorLog,
	}

	group.Go(func() error {
		err := metricsServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	healthOnlyRouter := http.NewServeMux()
	healthOnlyRouter.HandleFunc("/healthz", healthHandler)

	plaintextServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.HTTP,
		Handler:           healthOnlyRouter,
		ErrorLog:          httpErrorLog,
	}

	group.Go(func() error {
		err := plaintextServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	authn := newAuthenticator(options)
	router.HandleFunc("/", proxyMiddleware(proxy, authn, k8s.Config.BearerToken))
	tlsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.HTTPS,
		TLSConfig:         tlsConfig,
		Handler:           router,
		ErrorLog:          httpErrorLog,
	}

	logging.Infof("starting infra connector (%s) - http:%s https:%s metrics:%s", internal.FullVersion(), plaintextServer.Addr, tlsServer.Addr, metricsServer.Addr)

	group.Go(func() error {
		err = tlsServer.ListenAndServeTLS("", "")
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	// wait for shutdown signal
	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer shutdownCancel()
	if err := tlsServer.Shutdown(shutdownCtx); err != nil {
		logging.L.Warn().Err(err).Msgf("shutdown proxy server")
	}
	if err := plaintextServer.Close(); err != nil {
		logging.L.Warn().Err(err).Msgf("shutdown plaintext server")
	}
	if err := metricsServer.Close(); err != nil {
		logging.L.Warn().Err(err).Msgf("shutdown metrics server")
	}

	// wait for goroutines to shutdown
	err = group.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func healthHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		resp.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	resp.WriteHeader(http.StatusOK)
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

func syncDestination(ctx context.Context, con connector) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	endpoint, err := getEndpointHostPort(con.k8s, con.options)
	if err != nil {
		logging.L.Warn().Err(err).Msg("could not get host")
	}

	if endpoint.Host != "" {
		if ipAddr := net.ParseIP(endpoint.Host); ipAddr == nil {
			// wait for DNS resolution if endpoint is not an IP address
			if _, err := net.LookupIP(endpoint.Host); err != nil {
				logging.L.Warn().Err(err).Msg("host could not be resolved")
				endpoint.Host = ""
			}
		}

		// update certificates if the host changed
		_, err = con.certCache.AddHost(endpoint.Host)
		if err != nil {
			return fmt.Errorf("could not update self-signed certificates: %w", err)
		}

		logging.L.Debug().Str("addr", endpoint.String()).Msg("connector endpoint address")
	}

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

	case con.destination.Connection.URL != endpoint.String():
		con.destination.Connection.URL = endpoint.String()

		if err := createOrUpdateDestination(ctx, con.client, con.destination); err != nil {
			return fmt.Errorf("create or update destination: %w", err)
		}
	}
	return nil
}

func getEndpointHostPort(k8s kubeClient, opts Options) (types.HostPort, error) {
	if opts.EndpointAddr.Host != "" {
		return opts.EndpointAddr, nil
	}

	host, port, err := k8s.Endpoint()
	if err != nil {
		return types.HostPort{}, fmt.Errorf("failed to lookup endpoint: %w", err)
	}

	return types.HostPort{Host: host, Port: port}, nil
}

type waiter interface {
	Reset()
	Wait(ctx context.Context) error
}

func syncGrantsToDestination(
	ctx context.Context,
	con connector,
	waiter waiter,
	toDestination func(context.Context, []api.Grant) error,
) error {
	var latestIndex int64 = 1

	sync := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 7*time.Minute)
		defer cancel()

		grants, err := con.client.ListGrants(ctx, api.ListGrantsRequest{
			Destination:     con.destination.Name, // TODO: use options.Name when that is required
			BlockingRequest: api.BlockingRequest{LastUpdateIndex: latestIndex},
		})
		var apiError api.Error
		switch {
		case errors.As(err, &apiError) && apiError.Code == http.StatusNotModified:
			// not modified is expected when there are no changes
			logging.L.Info().
				Int64("updateIndex", latestIndex).
				Msg("no updated grants from server")
			return nil
		case err != nil:
			return fmt.Errorf("list grants: %w", err)
		}
		logging.L.Info().
			Int64("updateIndex", grants.LastUpdateIndex.Index).
			Int("grants", len(grants.Items)).
			Msg("received grants from server")

		err = toDestination(ctx, grants.Items)
		if err != nil {
			return fmt.Errorf("sync to destination: %w", err)
		}

		// Only update latestIndex once the entire operation was a success
		latestIndex = grants.LastUpdateIndex.Index
		return nil
	}

	for {
		if err := sync(ctx); err != nil {
			logging.L.Error().Err(err).Msg("sync grants to destination")
		} else {
			waiter.Reset()
		}

		// sleep for a short duration between updates to allow batches of
		// updates to apply before querying again, and to prevent unnecessary
		// load when part of the operation is failing.
		if err := waiter.Wait(ctx); err != nil {
			return err
		}
	}
}

// UpdateRoles converts infra grants to role-bindings in the current cluster
func updateRoles(ctx context.Context, c apiClient, k kubeClient, grants []api.Grant) error {
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
			group, err := c.GetGroup(ctx, g.Group)
			if err != nil {
				return err
			}

			name = group.Name
			kind = rbacv1.GroupKind
		case g.User != 0:
			user, err := c.GetUser(ctx, g.User)
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
		return fmt.Errorf("update role bindings: %w", err)
	}

	return nil
}

// createOrUpdateDestination creates a destination in the infra server if it does not exist and updates it if it does
func createOrUpdateDestination(ctx context.Context, client apiClient, local *api.Destination) error {
	// TODO: we probably don't want to cache the ID
	if local.ID != 0 {
		return updateDestination(ctx, client, local)
	}

	destinations, err := client.ListDestinations(ctx, api.ListDestinationsRequest{
		Name: local.Name,
	})
	if err != nil {
		return fmt.Errorf("error listing destinations: %w", err)
	}

	if destinations.Count > 0 {
		local.ID = destinations.Items[0].ID
		return updateDestination(ctx, client, local)
	}

	request := &api.CreateDestinationRequest{
		Name:       local.Name,
		Kind:       local.Kind,
		UniqueID:   local.UniqueID,
		Version:    internal.FullVersion(),
		Connection: local.Connection,
		Resources:  local.Resources,
		Roles:      local.Roles,
	}

	destination, err := client.CreateDestination(ctx, request)
	if err != nil {
		return fmt.Errorf("error creating destination: %w", err)
	}

	local.ID = destination.ID
	return nil
}

// updateDestination updates a destination in the infra server
func updateDestination(ctx context.Context, client apiClient, local *api.Destination) error {
	logging.Debugf("updating destination details")

	request := api.UpdateDestinationRequest{
		ID:         local.ID,
		Name:       local.Name,
		UniqueID:   local.UniqueID,
		Version:    internal.FullVersion(),
		Connection: local.Connection,
		Resources:  local.Resources,
		Roles:      local.Roles,
	}

	if _, err := client.UpdateDestination(ctx, request); err != nil {
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
