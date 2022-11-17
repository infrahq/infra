package k8s2

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/connector"
	"github.com/infrahq/infra/uid"
)

type k8sConnector struct {
	k8s         kubeClient
	client      apiClient
	destination *api.Destination
	certCache   *connector.CertCache
	options     connector.Options
}

type listCredentialRequest struct {
	api.BlockingRequest
}

type credentialRequest struct {
	////////////////////
	// request fields //
	////////////////////
	ID     uid.ID
	UserID uid.ID

	ExpiresAt time.Time

	// certificate
	PublicCertificate []byte // supplied if the user is planning to connect via client-generated certificate pair

	// ssh
	PublicKey []byte // supplied if the user is planning to connect via client-generated key pair

	/////////////////////
	// response fields //
	/////////////////////

	// username & pw
	Username string
	Password string

	// API key
	BearerToken string

	// Certificate

	// JWT or generic headers
	HeaderName string
	Token      string

	//...
}

// type apiClient interface {
// 	ListGrants(ctx context.Context, req api.ListGrantsRequest) (*api.ListResponse[api.Grant], error)
// 	ListCredentialRequests(ctx context.Context, req listCredentialRequest) (*api.ListResponse[credentialRequest], error)

// 	// GetGroup and GetUser are used to retrieve the name of the group or user.
// 	// TODO: we can remove these calls to GetGroup and GetUser by including
// 	// the name of the group or user in the ListGrants response.
// 	GetGroup(ctx context.Context, id uid.ID) (*api.Group, error)
// 	GetUser(ctx context.Context, id uid.ID) (*api.User, error)
// }

// type kubeClient interface {
// 	// Namespaces() ([]string, error)
// 	// ClusterRoles() ([]string, error)
// 	// IsServiceTypeClusterIP() (bool, error)
// 	// Endpoint() (string, int, error)

// 	UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error
// 	UpdateRoleBindings(subjects map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) error
// }

// func Run(ctx context.Context, options connector.Options) error {
// 	k8s, err := kubernetes.NewKubernetes(
// 		options.Kubernetes.AuthToken.String(),
// 		options.Kubernetes.Addr,
// 		options.Kubernetes.CA.String())
// 	if err != nil {
// 		return fmt.Errorf("failed to create kubernetes client: %w", err)
// 	}

// 	checkSum := k8s.Checksum()
// 	logging.L.Debug().Str("uniqueID", checkSum).Msg("Cluster uniqueID")

// 	if options.Name == "" {
// 		autoname, err := k8s.Name(checkSum)
// 		if err != nil {
// 			logging.Errorf("k8s name error: %s", err)
// 			return err
// 		}
// 		options.Name = autoname
// 	}

// 	certCache := connector.NewCertCache([]byte(options.CACert), []byte(options.CAKey))

// 	u, err := urlx.Parse(options.Server.URL)
// 	if err != nil {
// 		return fmt.Errorf("invalid server url: %w", err)
// 	}

// 	u.Scheme = "https"
// 	destination := &api.Destination{
// 		Name:     options.Name,
// 		UniqueID: checkSum,
// 	}

// 	client := &api.Client{
// 		Name:      "connector",
// 		Version:   internal.Version,
// 		URL:       u.String(),
// 		AccessKey: options.Server.AccessKey.String(),
// 		HTTP: http.Client{
// 			Transport: httpTransportFromOptions(options.Server),
// 		},
// 		Headers: http.Header{
// 			"Infra-Destination": {checkSum},
// 		},
// 	}

// 	group, ctx := errgroup.WithContext(ctx)

// 	con := k8sConnector{
// 		k8s:         k8s,
// 		client:      client,
// 		destination: destination,
// 		certCache:   certCache,
// 		options:     options,
// 	}
// 	group.Go(func() error {
// 		backOff := &backoff.ExponentialBackOff{
// 			InitialInterval:     2 * time.Second,
// 			MaxInterval:         time.Minute,
// 			RandomizationFactor: 0.2,
// 			Multiplier:          1.5,
// 		}
// 		waiter := repeat.NewWaiter(backOff)
// 		return syncGrantsToKubeBindings(ctx, con, waiter)
// 	})

// 	group.Go(func() error {
// 		backOff := &backoff.ExponentialBackOff{
// 			InitialInterval:     2 * time.Second,
// 			MaxInterval:         time.Minute,
// 			RandomizationFactor: 0.2,
// 			Multiplier:          1.5,
// 		}
// 		waiter := repeat.NewWaiter(backOff)
// 		return watchForCredentialRequests(ctx, con, waiter)
// 	})

// 	// wait for shutdown signal
// 	<-ctx.Done()

// 	return err
// }

// type waiter interface {
// 	Reset()
// 	Wait(ctx context.Context) error
// }

// func syncGrantsToKubeBindings(ctx context.Context, con k8sConnector, waiter waiter) error {
// 	var latestIndex int64 = 1

// 	sync := func() error {
// 		grants, err := con.client.ListGrants(ctx, api.ListGrantsRequest{
// 			Destination:     con.destination.Name,
// 			BlockingRequest: api.BlockingRequest{LastUpdateIndex: latestIndex},
// 		})
// 		var apiError api.Error
// 		switch {
// 		case errors.As(err, &apiError) && apiError.Code == http.StatusNotModified:
// 			// not modified is expected when there are no changes
// 			logging.L.Info().
// 				Int64("updateIndex", latestIndex).
// 				Msg("no updated grants from server")
// 			return nil
// 		case err != nil:
// 			return fmt.Errorf("list grants: %w", err)
// 		}
// 		logging.L.Info().
// 			Int64("updateIndex", latestIndex).
// 			Int("grants", len(grants.Items)).
// 			Msg("received grants from server")

// 		err = updateRoles(ctx, con.client, con.k8s, grants.Items)
// 		if err != nil {
// 			return fmt.Errorf("update roles: %w", err)
// 		}

// 		// Only update latestIndex once the entire operation was a success
// 		latestIndex = grants.LastUpdateIndex.Index
// 		return nil
// 	}

// 	for {
// 		if err := sync(); err != nil {
// 			logging.L.Error().Err(err).Msg("sync grants with kubernetes")
// 		} else {
// 			waiter.Reset()
// 		}

// 		// sleep for a short duration between updates to allow batches of
// 		// updates to apply before querying again, and to prevent unnecessary
// 		// load when part of the operation is failing.
// 		if err := waiter.Wait(ctx); err != nil {
// 			return err
// 		}
// 	}
// }

// // UpdateRoles converts infra grants to role-bindings in the current cluster
// func updateRoles(ctx context.Context, c apiClient, k kubeClient, grants []api.Grant) error {
// 	logging.Debugf("syncing local grants from infra configuration")

// 	crSubjects := make(map[string][]rbacv1.Subject)                           // cluster-role: subject
// 	crnSubjects := make(map[kubernetes.ClusterRoleNamespace][]rbacv1.Subject) // cluster-role+namespace: subject

// 	for _, g := range grants {
// 		var name, kind string

// 		if g.Privilege == "connect" {
// 			continue
// 		}

// 		switch {
// 		case g.Group != 0:
// 			group, err := c.GetGroup(ctx, g.Group)
// 			if err != nil {
// 				return err
// 			}

// 			name = group.Name
// 			kind = rbacv1.GroupKind
// 		case g.User != 0:
// 			user, err := c.GetUser(ctx, g.User)
// 			if err != nil {
// 				return err
// 			}

// 			name = user.Name
// 			kind = rbacv1.UserKind
// 		}

// 		subj := rbacv1.Subject{
// 			APIGroup: "rbac.authorization.k8s.io",
// 			Kind:     kind,
// 			Name:     name,
// 		}

// 		parts := strings.Split(g.Resource, ".")

// 		var crn kubernetes.ClusterRoleNamespace

// 		switch len(parts) {
// 		// <cluster>
// 		case 1:
// 			crn.ClusterRole = g.Privilege
// 			crSubjects[g.Privilege] = append(crSubjects[g.Privilege], subj)

// 		// <cluster>.<namespace>
// 		case 2:
// 			crn.ClusterRole = g.Privilege
// 			crn.Namespace = parts[1]
// 			crnSubjects[crn] = append(crnSubjects[crn], subj)

// 		default:
// 			logging.Warnf("invalid grant resource: %s", g.Resource)
// 			continue
// 		}
// 	}

// 	if err := k.UpdateClusterRoleBindings(crSubjects); err != nil {
// 		return fmt.Errorf("update cluster role bindings: %w", err)
// 	}

// 	if err := k.UpdateRoleBindings(crnSubjects); err != nil {
// 		return fmt.Errorf("update role bindings: %w", err)
// 	}

// 	return nil
// }

// func httpTransportFromOptions(opts connector.ServerOptions) *http.Transport {
// 	roots, err := x509.SystemCertPool()
// 	if err != nil {
// 		logging.L.Warn().Err(err).Msgf("failed to load TLS roots from system")
// 		roots = x509.NewCertPool()
// 	}

// 	if opts.TrustedCertificate != "" {
// 		if !roots.AppendCertsFromPEM([]byte(opts.TrustedCertificate)) {
// 			logging.Warnf("failed to load TLS CA, invalid PEM")
// 		}
// 	}

// 	// nolint:forcetypeassert // http.DefaultTransport is always http.Transport
// 	transport := http.DefaultTransport.(*http.Transport).Clone()
// 	transport.TLSClientConfig = &tls.Config{
// 		//nolint:gosec // We may purposely set InsecureSkipVerify via a flag
// 		InsecureSkipVerify: opts.SkipTLSVerify,
// 		RootCAs:            roots,
// 	}
// 	return transport
// }

// const maxConcurrentRequests = 50

// func watchForCredentialRequests(ctx context.Context, con k8sConnector, waiter waiter) error {
// 	lastUpdateIndex := int64(0)
// 	workerPool := semaphore.NewWeighted(maxConcurrentRequests)
// 	for {
// 		resp, err := con.client.ListCredentialRequests(ctx, listCredentialRequest{BlockingRequest: api.BlockingRequest{LastUpdateIndex: lastUpdateIndex}})
// 		if err != nil {
// 			logging.Errorf("watchForCredentialRequests: %s", err)
// 			waiter.Wait(ctx)
// 			continue
// 		}
// 		waiter.Reset()

// 		for i := range resp.Items {
// 			err := workerPool.Acquire(ctx, 1)
// 			if err != nil {
// 				return err
// 			}

// 			go func() {
// 				err := serveCredentialRequest(ctx, con, resp.Items[i])
// 				if err != nil {
// 					logging.Errorf("serveCredentialRequest: %s", err)
// 				}
// 				workerPool.Release(1)
// 			}()
// 		}
// 	}
// 	return nil
// }

// func serveCredentialRequest(ctx context.Context, con k8sConnector, req credentialRequest) error {
// 	req.UserID
// 	return nil
// }
