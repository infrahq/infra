package k8s3

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
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

type Options struct {
	Server ServerOptions
	Name   string
	CACert types.StringOrFile
	CAKey  types.StringOrFile

	Addr ListenerOptions
	// EndpointAddr is the host:port address where the connector proxy receives
	// requests to the destination. If this value is empty then the host:port
	// will be looked up from the kube API, using the name of the connector
	// service.
	// This value is sent to the infra API server to update the
	// Destination.Connection.URL.
	EndpointAddr types.HostPort

	Kubernetes KubernetesOptions
}

type ServerOptions struct {
	URL                string
	AccessKey          types.StringOrFile
	SkipTLSVerify      bool
	TrustedCertificate types.StringOrFile
}

type ListenerOptions struct {
	HTTP    string
	HTTPS   string
	Metrics string
}

type KubernetesOptions struct {
	// AuthToken may be used to override the token used to authenticate with the
	// kubernetes API server. When the connector is run in-cluster, the
	// service account associated with the pod will be used by default.
	// When run outside of cluster there is no default, and this value must
	// be set to a token that has permission to impersonate users in the cluster.
	AuthToken types.StringOrFile

	// Addr is the host:port used to connect to the kubernetes API server. The
	// default value is looked up from the in-cluster config.
	Addr string
	// CA is the CA certificate used by the kubernetes API server. The default
	// value is looked up from the in-cluster config.
	CA types.StringOrFile
}

// k8sConnector stores all the dependencies for the k8sConnector operations.
type k8sConnector struct {
	k8s         kubeClient
	client      apiClient
	destination api.Destination
	options     Options
}

type apiClient interface {
	ListGrants(ctx context.Context, req api.ListGrantsRequest) (*api.ListResponse[api.Grant], error)
	ListDestinations(ctx context.Context, req api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error)
	CreateDestination(ctx context.Context, req *api.CreateDestinationRequest) (*api.Destination, error)
	UpdateDestination(ctx context.Context, req api.UpdateDestinationRequest) (*api.Destination, error)

	ListCredentialRequests(ctx context.Context, req api.ListCredentialRequest) (*api.ListCredentialRequestResponse, error)
	UpdateCredentialRequest(ctx context.Context, r *api.UpdateCredentialRequest) (*api.EmptyResponse, error)

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
	DirectEndpoint() (string, int, []byte, error)

	UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error
	UpdateRoleBindings(subjects map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) error
	CreateServiceAccount(username string) (*corev1.ServiceAccount, error)
	CreateServiceAccountToken(username string) (*authenticationv1.TokenRequest, error)
}

var httpErrorLog *log.Logger

func Run(ctx context.Context, options Options) error {
	logging.Debugf("k8s3 started")

	httpErrorLog = log.New(logging.NewFilteredHTTPLogger(), "", 0)

	u, err := urlx.Parse(options.Server.URL)
	if err != nil {
		return fmt.Errorf("invalid server url: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	u.Scheme = "https"

	runGroup, ctx := errgroup.WithContext(ctx)

	con1, err := newK8sConnector(options)
	if err != nil {
		return err
	}
	con2, err := newK8sConnector(options)
	if err != nil {
		return err
	}
	con3, err := newK8sConnector(options)
	if err != nil {
		return err
	}
	con4, err := newK8sConnector(options)
	if err != nil {
		return err
	}

	// sync destination once to populate the con's destination fields. these get shared across goroutines.
	err = syncDestination(ctx, con3)
	if err != nil {
		return err
	}

	logging.Debugf("k8s3 starting services")

	startMetricsService(ctx, con1, runGroup)
	startHealthService(ctx, options, runGroup)
	startGrantReconciler(ctx, con2, runGroup)
	startDestinationSync(ctx, con3, runGroup)
	startCredentialRequestReconciler(ctx, con4, runGroup)

	logging.Debugf("starting infra k8s connector (%s)", internal.FullVersion())

	// wait for shutdown signal
	<-ctx.Done()

	logging.Debugf("shutdown received")

	// wait for goroutines to shutdown
	err = runGroup.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	logging.Errorf("rungroup error: %s", err)
	return err
}

func startCredentialRequestReconciler(ctx context.Context, con *k8sConnector, runGroup *errgroup.Group) {
	runGroup.Go(func() error {
		lastUpdateIndex := int64(0)
		for ctx.Err() == nil {
			resp := pollForCredentialRequestsOnce(ctx, con, lastUpdateIndex)
			if resp != nil {
				lastUpdateIndex = resp.MaxUpdateIndex
			}
		}
		logging.Debugf("CredentialRequestReconciler exited")
		return nil
	})
}

func pollForCredentialRequestsOnce(ctx context.Context, con *k8sConnector, lastUpdateIndex int64) *api.ListCredentialRequestResponse {
	client := con.client

	// long-poll credentials against server
	logging.Debugf("ListCredentialRequests since %d", lastUpdateIndex)
	resp, err := client.ListCredentialRequests(ctx, api.ListCredentialRequest{
		Destination:     con.destination.Name,
		LastUpdateIndex: lastUpdateIndex,
	})
	if err != nil {
		logging.Errorf("ListCredentialRequest: %s", err.Error())
		time.Sleep(1 * time.Second) // don't want to wait long as this is critical path
		return nil
	}
	logging.Debugf("ListCredentialRequests found %d requests", len(resp.Items))

	// for each: create service account
	for _, req := range resp.Items {
		user, err := con.client.GetUser(ctx, req.UserID)
		if err != nil {
			logging.Errorf("GetUser failed, ignoring credential request: %s", err.Error())
			return &api.ListCredentialRequestResponse{MaxUpdateIndex: req.UpdateIndex}
		}
		// issue token to user
		tokenRequest, err := con.k8s.CreateServiceAccountToken(normalizeUsername(user.Name))
		if err != nil {
			logging.Errorf("CreateServiceAccountToken failed, ignoring credential request: %s", err.Error())
			return &api.ListCredentialRequestResponse{MaxUpdateIndex: req.UpdateIndex}
		}
		// return credentials to user by posting back to the api.
		logging.Debugf("UpdateCredentialRequest %d for org %d", req.ID, req.OrganizationID)
		expiresAt := time.Now().Add(24 * time.Hour)
		if tokenRequest.Spec.ExpirationSeconds != nil {
			expiresAt = time.Now().Add(time.Duration(*(tokenRequest.Spec.ExpirationSeconds)) * time.Second)
		}
		_, err = con.client.UpdateCredentialRequest(ctx, &api.UpdateCredentialRequest{
			ID:             req.ID,
			OrganizationID: req.OrganizationID,
			BearerToken:    tokenRequest.Status.Token,
			ExpiresAt:      api.Time(expiresAt),
		})
		if err != nil {
			logging.Errorf("UpdateCredentialRequest failed, ignoring credential request: %s", err.Error())
			return &api.ListCredentialRequestResponse{MaxUpdateIndex: req.UpdateIndex}
		}
	}

	return resp
}

func newK8sConnector(options Options) (*k8sConnector, error) {
	k8s, err := kubernetes.NewKubernetes(
		options.Kubernetes.AuthToken.String(),
		options.Kubernetes.Addr,
		options.Kubernetes.CA.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	checkSum := k8s.Checksum()
	logging.L.Debug().Str("uniqueID", checkSum).Msg("Cluster uniqueID")

	if options.Name == "" {
		autoname, err := k8s.Name(checkSum)
		if err != nil {
			logging.Errorf("k8s name error: %s", err)
			return nil, err
		}
		options.Name = autoname
	}

	destination := api.Destination{
		Name:     options.Name,
		UniqueID: checkSum,
	}

	u, err := urlx.Parse(options.Server.URL)
	if err != nil {
		return nil, err
	}

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
			logging.Errorf("Unauthorized error; token invalid or expired")
		},
	}

	return &k8sConnector{
		destination: destination,
		client:      client,
		k8s:         k8s,
		options:     options,
	}, nil
}

func startGrantReconciler(ctx context.Context, con *k8sConnector, runGroup *errgroup.Group) {
	runGroup.Go(func() error {
		backOff := &backoff.ExponentialBackOff{
			InitialInterval:     2 * time.Second,
			MaxInterval:         time.Minute,
			RandomizationFactor: 0.2,
			Multiplier:          1.5,
		}
		waiter := repeat.NewWaiter(backOff)
		defer logging.Debugf("GrantReconciler exited")
		return syncGrantsToKubeBindings(ctx, con, waiter)
	})
}

func startDestinationSync(ctx context.Context, con *k8sConnector, runGroup *errgroup.Group) {
	runGroup.Go(func() error {
		waiter := repeat.NewWaiter(backoff.NewConstantBackOff(30 * time.Second))
		tick := time.NewTicker(5 * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := syncDestination(ctx, con); err != nil {
					logging.Errorf("failed to update destination in infra: %v", err)
					if err := waiter.Wait(ctx); err != nil {
						return err
					}
				} else {
					waiter.Reset()
				}
			case <-ctx.Done():
				logging.Debugf("DestinationSync exited")
				return nil
			}
		}
	})
}

func startHealthService(ctx context.Context, options Options, runGroup *errgroup.Group) {
	gin.SetMode(gin.ReleaseMode)
	healthOnlyRouter := gin.New()
	healthOnlyRouter.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	plaintextServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              options.Addr.HTTP,
		Handler:           healthOnlyRouter,
		ErrorLog:          httpErrorLog,
	}

	runGroup.Go(func() error {
		err := plaintextServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	go func() {
		<-ctx.Done()
		if err := plaintextServer.Close(); err != nil {
			logging.L.Warn().Err(err).Msgf("shutdown plaintext server")
		}
		logging.Debugf("HealthService exited")
	}()
}

func startMetricsService(ctx context.Context, con *k8sConnector, runGroup *errgroup.Group) {
	promRegistry := metrics.NewRegistry(internal.FullVersion())
	responseDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http_client",
		Name:      "request_duration_seconds",
		Help:      "A histogram of duration, in seconds, performing HTTP requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"host", "method", "path", "status"})
	promRegistry.MustRegister(responseDuration)

	if c, ok := con.client.(*api.Client); ok {
		c.ObserveFunc = func(start time.Time, request *http.Request, response *http.Response, err error) {
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
	}

	metricsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              con.options.Addr.Metrics,
		Handler:           metrics.NewHandler(promRegistry),
		ErrorLog:          httpErrorLog,
	}

	runGroup.Go(func() error {
		err := metricsServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})

	go func() {
		<-ctx.Done()
		if err := metricsServer.Close(); err != nil {
			logging.L.Warn().Err(err).Msgf("shutdown metrics server")
		}
		logging.Debugf("MetricsService exited")
	}()
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

// syncDestination creates and manages the destination record in Infra.
func syncDestination(ctx context.Context, con *k8sConnector) error {
	endpoint, cert, err := getEndpointHostPort(con.k8s, con.options)
	if err != nil {
		logging.L.Warn().Err(err).Msg("could not get host")
	}

	if cert != nil {
		con.options.CACert = types.StringOrFile(cert)
	}

	if endpoint.Host != "" {
		if ipAddr := net.ParseIP(endpoint.Host); ipAddr == nil {
			// wait for DNS resolution if endpoint is not an IP address
			if _, err := net.LookupIP(endpoint.Host); err != nil {
				logging.L.Warn().Err(err).Msg("host could not be resolved")
			}
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

		if err := createOrUpdateDestination(ctx, con.client, &con.destination); err != nil {
			return fmt.Errorf("create or update destination: %w", err)
		}
	}
	return nil
}

func getEndpointHostPort(k8s kubeClient, opts Options) (types.HostPort, []byte, error) {
	if opts.EndpointAddr.Host != "" {
		return opts.EndpointAddr, nil, nil
	}

	host, port, cert, err := k8s.DirectEndpoint()
	if err != nil {
		return types.HostPort{}, nil, fmt.Errorf("failed to lookup endpoint: %w", err)
	}

	return types.HostPort{Host: host, Port: port}, cert, nil
}

type waiter interface {
	Reset()
	Wait(ctx context.Context) error
}

// syncGrantsToKubeBindings reconciles grants to the destination system (this piece is specific to this connector)
func syncGrantsToKubeBindings(ctx context.Context, con *k8sConnector, waiter waiter) error {
	var latestIndex int64 = 1

	options := con.options
	u, err := urlx.Parse(options.Server.URL)
	if err != nil {
		return err
	}

	client := &api.Client{
		Name:      "connector",
		Version:   internal.Version,
		URL:       u.String(),
		AccessKey: options.Server.AccessKey.String(),
		HTTP: http.Client{
			Transport: httpTransportFromOptions(options.Server),
		},
		Headers: http.Header{
			"Infra-Destination": {con.destination.UniqueID},
		},
		OnUnauthorized: func() {
			logging.Errorf("Unauthorized error; token invalid or expired. exiting.")
		},
	}

	sync := func() error {
		grants, err := client.ListGrants(ctx, api.ListGrantsRequest{
			Destination:     con.destination.Name,
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
			Int64("updateIndex", latestIndex).
			Int("grants", len(grants.Items)).
			Msg("received grants from server")

		err = updateRoles(ctx, con, grants.Items)
		if err != nil {
			return fmt.Errorf("update roles: %w", err)
		}

		// Only update latestIndex once the entire operation was a success
		latestIndex = grants.LastUpdateIndex.Index
		return nil
	}

	for {
		if err := sync(); err != nil {
			logging.L.Error().Err(err).Msg("sync grants with kubernetes")
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
func updateRoles(ctx context.Context, con *k8sConnector, grants []api.Grant) error {
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
			group, err := con.client.GetGroup(ctx, g.Group)
			if err != nil {
				return err
			}

			name = group.Name
			kind = rbacv1.GroupKind
		case g.User != 0:
			user, err := con.client.GetUser(ctx, g.User)
			if err != nil {
				return err
			}

			name = normalizeUsername(user.Name)
			kind = rbacv1.ServiceAccountKind
		}
		subj := rbacv1.Subject{
			Kind:      kind,
			Name:      name,
			Namespace: "default",
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

	if err := con.k8s.UpdateClusterRoleBindings(crSubjects); err != nil {
		return fmt.Errorf("update cluster role bindings: %w", err)
	}

	if err := con.k8s.UpdateRoleBindings(crnSubjects); err != nil {
		return fmt.Errorf("update role bindings: %w", err)
	}

	return nil
}

// createOrUpdateDestination creates a destination in the infra server if it does not exist and updates it if it does
func createOrUpdateDestination(ctx context.Context, client apiClient, local *api.Destination) error {
	if local.ID != 0 {
		return updateDestination(ctx, client, local)
	}

	destinations, err := client.ListDestinations(ctx, api.ListDestinationsRequest{UniqueID: local.UniqueID})
	if err != nil {
		return fmt.Errorf("error listing destinations: %w", err)
	}

	if destinations.Count > 0 {
		local.ID = destinations.Items[0].ID
		return updateDestination(ctx, client, local)
	}

	request := &api.CreateDestinationRequest{
		Name:       local.Name,
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

const validCharacters string = "qwertyuiopasdfghjklzxcvbnm1234567890-."

func normalizeUsername(name string) string {
	result := ""
	for i := range strings.ToLower(name) {
		if strings.ContainsAny(string(name[i]), validCharacters) {
			result += string(name[i])
		} else {
			result += "."
		}
	}
	result = strings.ReplaceAll(result, "..", ".")
	result = strings.ReplaceAll(result, "--", "-")
	parts := strings.Split(result, ".")
	acceptedParts := []string{}
	// parts must not start or end with non-alphanumerics
	for _, part := range parts {
		for len(part) > 0 && (part[0] < 'a' || part[0] > 'z') && (part[0] < '0' || part[0] > '9') {
			part = part[1:]
		}
		for len(part) > 0 && (part[len(part)-1] < 'a' || part[len(part)-1] > 'z') && (part[len(part)-1] < '0' || part[len(part)-1] > '9') {
			part = part[:len(part)-2]
		}
		if len(part) > 0 {
			acceptedParts = append(acceptedParts, part)
		}
	}
	full := strings.Join(acceptedParts, ".")
	return full
}
