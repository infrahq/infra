package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/data/encrypt"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/internal/server/redis"
	"github.com/infrahq/infra/metrics"
	"github.com/infrahq/infra/uid"
)

type Options struct {
	Version  float64
	TLSCache string // TODO: move this to TLS.CacheDir

	EnableTelemetry bool

	// EnableSignup indicates that anyone can signup and create an org. When
	// true this implies multi-tenancy, but false does not necessarily indicate
	// a single tenancy environment (because orgs could have been created by a
	// support admin).
	EnableSignup bool

	// EnableLogSampling indicates whether or not to sample HTTP access logs.
	// When true, non-error HTTP GET logs will sampled down to 1 every 7 seconds
	// grouped by the request path.
	EnableLogSampling bool

	SessionDuration          time.Duration // the lifetime of the access key infra issues on login
	SessionInactivityTimeout time.Duration // access keys issued on login must be used within this window of time, or they become invalid

	// Redis contains configuration options to the cache server.
	Redis redis.Options

	GoogleClientID     string
	GoogleClientSecret string

	DBEncryptionKey    string
	DBHost             string
	DBPort             int
	DBName             string
	DBUsername         string
	DBPassword         string
	DBParameters       string
	DBConnectionString string

	EmailAppDomain   string
	EmailFromAddress string
	EmailFromName    string
	SendgridApiKey   string
	SMTPServer       string

	// BaseDomain of the server, which is appended to the organization slug to
	// create a unique hostname for each organization.
	BaseDomain string
	// LoginDomainPrefix that users will be sent to after logging in with Google
	LoginDomainPrefix string

	BootstrapConfig

	Addr ListenerOptions
	UI   UIOptions
	TLS  TLSOptions
	API  APIOptions

	DB data.NewDBOptions

	DeprecatedConfig
}

// DeprecatedConfig contains fields that are no longer used by server, but loading
// values for these fields allows us to error when a config file value is no
// longer supported.
type DeprecatedConfig struct {
	DBEncryptionKeyProvider string
	Providers               any
	Grants                  any
}

type ListenerOptions struct {
	HTTP    string
	HTTPS   string
	Metrics string
}

type UIOptions struct {
	ProxyURL types.URL
}

type TLSOptions struct {
	// CA is a PEM encoded certificate for the CA that signed the
	// certificate, or that will be used to generate a certificate if one was
	// not provided.
	CA           types.StringOrFile
	CAPrivateKey types.StringOrFile
	Certificate  types.StringOrFile
	PrivateKey   types.StringOrFile

	// ACME enables automated certificate management. When set to true a TLS
	// certificate will be requested from Let's Encrypt, which will be cached
	// in the TLSCache.
	ACME bool
}

type APIOptions struct {
	RequestTimeout         time.Duration
	BlockingRequestTimeout time.Duration
}

type Server struct {
	options         Options
	db              *data.DB
	redis           *redis.Redis
	tel             *Telemetry
	Addrs           Addrs
	routines        []routine
	metricsRegistry *prometheus.Registry
	Google          *models.Provider
}

type Addrs struct {
	HTTP    net.Addr
	HTTPS   net.Addr
	Metrics net.Addr
}

// newServer creates a Server with base dependencies initialized to zero values.
func newServer(options Options) *Server {
	return &Server{options: options}
}

// New creates a Server, and initializes it. The returned Server is ready to run.
func New(options Options) (*Server, error) {
	if options.EnableSignup && options.BaseDomain == "" {
		return nil, errors.New("cannot enable signup without setting base domain")
	}

	if options.DBEncryptionKeyProvider != "" && options.DBEncryptionKeyProvider != "native" {
		return nil, errors.New("dbEncryptionKeyProvider is no longer supported, " +
			"use a file for the root key and set dbEncryptionKey to the path of the file")
	}
	if options.Grants != nil {
		return nil, fmt.Errorf("grants can no longer be defined from config. " +
			"Please use https://github.com/infrahq/terraform-provider-infra or the API")
	}
	if options.Providers != nil {
		return nil, fmt.Errorf("providers can no longer be defined from config. " +
			"Please use https://github.com/infrahq/terraform-provider-infra or the API")
	}

	server := newServer(options)

	dsn, err := getPostgresConnectionString(options)
	if err != nil {
		return nil, fmt.Errorf("postgres dsn: %w", err)
	}
	options.DB.DSN = dsn
	options.DB.RootKeyFilePath = options.DBEncryptionKey

	if _, err := os.Stat(options.DB.RootKeyFilePath); errors.Is(err, fs.ErrNotExist) {
		if err := encrypt.CreateRootKey(options.DB.RootKeyFilePath); err != nil {
			return nil, err
		}
	}

	db, err := data.NewDB(options.DB)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	server.db = db
	server.metricsRegistry = setupMetrics(server.db)

	server.redis, err = redis.NewRedis(options.Redis)
	if err != nil {
		return nil, err
	}

	if options.EnableTelemetry {
		server.tel = NewTelemetry(server.db, db.DefaultOrgSettings.ID)
	}

	if options.GoogleClientID != "" {
		server.Google = &models.Provider{
			Model: models.Model{
				ID: models.InternalGoogleProviderID,
			},
			Name:         "Google",
			URL:          "accounts.google.com",
			ClientID:     options.GoogleClientID,
			ClientSecret: models.EncryptedAtRest(options.GoogleClientSecret),
			CreatedBy:    models.CreatedBySystem,
			Kind:         models.ProviderKindGoogle,
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			Scopes:       []string{"openid", "email"}, // TODO: update once our social client has groups
		}
	}

	if err := server.loadConfig(server.options.BootstrapConfig); err != nil {
		return nil, fmt.Errorf("configs: %w", err)
	}

	if err := server.listen(); err != nil {
		return nil, fmt.Errorf("listening: %w", err)
	}

	configureEmail(options)

	return server, nil
}

// DB returns an instance of a database connection pool that is used by the server.
// It is primarily used by tests to create fixture data.
func (s *Server) DB() *data.DB {
	return s.db
}

func (s *Server) Options() Options {
	return s.options
}

func (s *Server) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	group.Go(backgroundJob(ctx, s.db, data.DeleteExpiredDeviceFlowAuthRequests, 10*time.Minute))
	group.Go(backgroundJob(ctx, s.db, data.RemoveExpiredAccessKeys, 12*time.Hour))
	group.Go(backgroundJob(ctx, s.db, data.RemoveExpiredPasswordResetTokens, 15*time.Minute))
	group.Go(backgroundJob(ctx, s.db, data.DeleteExpiredUserPublicKeys, time.Hour))

	if s.tel != nil {
		group.Go(func() error {
			return runTelemetryHeartbeat(ctx, s.tel)
		})
	}

	for i := range s.routines {
		group.Go(s.routines[i].run)
	}

	logging.Infof("starting infra server (%s) - http:%s https:%s metrics:%s",
		internal.FullVersion(), s.Addrs.HTTP, s.Addrs.HTTPS, s.Addrs.Metrics)

	<-ctx.Done()
	for i := range s.routines {
		s.routines[i].stop()
	}

	err := group.Wait()
	s.tel.Close()

	if err := s.db.Close(); err != nil {
		logging.L.Warn().Err(err).Msg("failed to close database connection")
	}

	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func runTelemetryHeartbeat(ctx context.Context, tel *Telemetry) error {
	waiter := repeat.NewWaiter(backoff.NewConstantBackOff(time.Hour))
	for {
		tel.EnqueueHeartbeat()
		if err := waiter.Wait(ctx); err != nil {
			return err
		}
	}
}

func registerUIRoutes(router *gin.Engine, opts UIOptions) {
	if opts.ProxyURL.Host == "" {
		return
	}
	remote := opts.ProxyURL.Value()
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
	}
	proxy.ErrorLog = log.New(logging.NewFilteredHTTPLogger(), "", 0)

	router.Use(func(c *gin.Context) {
		// Don't proxy /api/* paths
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	})
}

func (s *Server) listen() error {
	ginutil.SetMode()
	router := s.GenerateRoutes()

	httpErrorLog := log.New(logging.NewFilteredHTTPLogger(), "", 0)
	metricsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              s.options.Addr.Metrics,
		Handler:           metrics.NewHandler(s.metricsRegistry),
		ErrorLog:          httpErrorLog,
	}

	var err error
	s.Addrs.Metrics, err = s.setupServer(metricsServer)
	if err != nil {
		return err
	}

	plaintextServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              s.options.Addr.HTTP,
		Handler:           router,
		ErrorLog:          httpErrorLog,
	}
	s.Addrs.HTTP, err = s.setupServer(plaintextServer)
	if err != nil {
		return err
	}

	tlsConfig, err := tlsConfigFromOptions(s.options.TLS)
	if err != nil {
		return fmt.Errorf("tls config: %w", err)
	}

	tlsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              s.options.Addr.HTTPS,
		TLSConfig:         tlsConfig,
		Handler:           router,
		ErrorLog:          httpErrorLog,
	}
	s.Addrs.HTTPS, err = s.setupServer(tlsServer)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) setupServer(server *http.Server) (net.Addr, error) {
	if server.Addr == "" {
		server.Addr = "127.0.0.1:"
	}
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, err
	}
	logging.Infof("listening on %s", l.Addr().String())

	s.routines = append(s.routines, routine{
		run: func() error {
			var err error
			if server.TLSConfig == nil {
				err = server.Serve(l)
			} else {
				err = server.ServeTLS(l, "", "")
			}
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
		stop: func() {
			_ = server.Close()
		},
	})
	return l.Addr(), nil
}

type routine struct {
	run  func() error
	stop func()
}

// getPostgresConnectionString parses postgres configuration options and returns the connection string
func getPostgresConnectionString(options Options) (string, error) {
	var pgConn strings.Builder
	pgConn.WriteString(options.DBConnectionString + " ")

	if options.DBHost != "" {
		// config has separate postgres parameters set, combine them into a connection DSN now
		fmt.Fprintf(&pgConn, "host=%s ", options.DBHost)
	}

	if options.DBUsername != "" {
		fmt.Fprintf(&pgConn, "user=%s ", options.DBUsername)
	}

	if options.DBPassword != "" {
		fmt.Fprintf(&pgConn, "password=%s ", options.DBPassword)
	}

	if options.DBPort > 0 {
		fmt.Fprintf(&pgConn, "port=%d ", options.DBPort)
	}

	if options.DBName != "" {
		fmt.Fprintf(&pgConn, "dbname=%s ", options.DBName)
	}

	// TODO: deprecate DBParameters now that we accept DBConnectionString
	if options.DBParameters != "" {
		fmt.Fprint(&pgConn, options.DBParameters)
	}

	return strings.TrimSpace(pgConn.String()), nil
}

func configureEmail(options Options) {
	if len(options.EmailAppDomain) > 0 {
		email.AppDomain = options.EmailAppDomain
	}
	if len(options.EmailFromAddress) > 0 {
		email.FromAddress = options.EmailFromAddress
	}
	if len(options.EmailFromName) > 0 {
		email.FromName = options.EmailFromName
	}
	if len(options.SendgridApiKey) > 0 {
		email.SendgridAPIKey = options.SendgridApiKey
	}
	if len(options.SMTPServer) > 0 {
		email.SMTPServer = options.SMTPServer
	}
}

// providerUserUpdateThreshold is the duration of time that must pass before a
// users session is attempted to be validated again with an external identity provider.
// This prevents hitting IDP rate limits.
const providerUserUpdateThreshold = 120 * time.Minute

var ErrSyncFailed = fmt.Errorf("user sync failed")

// syncIdentityInfo calls the identity provider used to authenticate this user session to update their current information
func (s *Server) syncIdentityInfo(ctx context.Context, tx *data.Transaction, identity *models.Identity, sessionProviderID uid.ID) error {
	var provider *models.Provider
	if s.Google != nil && sessionProviderID == s.Google.ID {
		provider = s.Google
	} else {
		var err error
		provider, err = data.GetProvider(tx, data.GetProviderOptions{
			ByID: sessionProviderID,
		})
		if err != nil {
			return fmt.Errorf("failed to get provider for user info: %w", err)
		}

		if provider.Kind == models.ProviderKindInfra {
			// no external verification needed
			logging.L.Trace().Msg("skipped verifying identity within infra provider, not required")
			return nil
		}
	}

	providerUser, err := data.GetProviderUser(tx, provider.ID, identity.ID)
	if err != nil {
		return fmt.Errorf("failed to get provider user to update: %w", err)
	}

	// if provider user was updated recently, skip checking this now to avoid hitting rate limits
	if time.Since(providerUser.LastUpdate) > providerUserUpdateThreshold {
		oidc, err := s.providerClient(ctx, provider, providerUser.RedirectURL)
		if err != nil {
			return fmt.Errorf("update provider client: %w", err)
		}

		// update current identity provider groups and account status
		_, err = data.SyncProviderUser(ctx, tx, providerUser, oidc)
		if err != nil {
			if errors.Is(err, internal.ErrBadGateway) {
				return err
			}

			logging.L.Info().Msg("user session expired, pruning keys created for this session")

			if nestedErr := data.DeleteAccessKeys(tx, data.DeleteAccessKeysOptions{ByIssuedForID: providerUser.IdentityID, ByProviderID: providerUser.ProviderID}); nestedErr != nil {
				logging.Errorf("failed to revoke invalid user session: %s", nestedErr)
			}

			if nestedErr := data.DeleteProviderUsers(tx, data.DeleteProviderUsersOptions{ByIdentityID: providerUser.IdentityID, ByProviderID: providerUser.ProviderID}); nestedErr != nil {
				logging.Errorf("failed to delete provider user: %s", nestedErr)
			}

			return fmt.Errorf("%w: %s", ErrSyncFailed, err)
		}

		providerUser.LastUpdate = time.Now().UTC()
		if err := data.UpdateProviderUser(tx, providerUser); err != nil {
			return fmt.Errorf("update idp user: %w", err)
		}
	}

	return nil
}

func (s *Server) providerClient(ctx context.Context, provider *models.Provider, redirectURL string) (providers.OIDCClient, error) {
	if c := providers.OIDCClientFromContext(ctx); c != nil {
		// oidc is added to the context during unit tests
		return c, nil
	}

	if provider.ID == models.InternalGoogleProviderID {
		// load the secret google information now, it is not set by default to avoid the possibility of returning this secret info externally
		provider = s.Google
	}

	return providers.NewOIDCClient(*provider, string(provider.ClientSecret), redirectURL), nil
}
