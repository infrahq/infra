package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gin-gonic/gin"
	"github.com/infrahq/secrets"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/redis"
	"github.com/infrahq/infra/metrics"
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

	SessionDuration          time.Duration
	SessionExtensionDeadline time.Duration

	// Redis contains configuration options to the cache server.
	Redis redis.Options

	DBEncryptionKey         string
	DBEncryptionKeyProvider string
	DBHost                  string
	DBPort                  int
	DBName                  string
	DBUsername              string
	DBPassword              string
	DBParameters            string
	DBConnectionString      string

	EmailAppDomain   string
	EmailFromAddress string
	EmailFromName    string
	SendgridApiKey   string
	SMTPServer       string

	// BaseDomain of the server, which is appended to the organization slug to
	// create a unique hostname for each organization.
	BaseDomain string

	Keys    []KeyProvider
	Secrets []SecretProvider

	Config

	Addr ListenerOptions
	UI   UIOptions
	TLS  TLSOptions
	API  APIOptions

	DB data.NewDBOptions
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
	CAPrivateKey string
	Certificate  types.StringOrFile
	PrivateKey   string

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
	secrets         map[string]secrets.SecretStorage
	keys            map[string]secrets.SymmetricKeyProvider
	Addrs           Addrs
	routines        []routine
	metricsRegistry *prometheus.Registry
}

type Addrs struct {
	HTTP    net.Addr
	HTTPS   net.Addr
	Metrics net.Addr
}

// newServer creates a Server with base dependencies initialized to zero values.
func newServer(options Options) *Server {
	return &Server{
		options: options,
		secrets: map[string]secrets.SecretStorage{},
		keys:    map[string]secrets.SymmetricKeyProvider{},
	}
}

// New creates a Server, and initializes it. The returned Server is ready to run.
func New(options Options) (*Server, error) {
	if options.EnableSignup && options.BaseDomain == "" {
		return nil, errors.New("cannot enable signup without setting base domain")
	}

	server := newServer(options)

	if err := importSecrets(options.Secrets, server.secrets); err != nil {
		return nil, fmt.Errorf("secrets config: %w", err)
	}

	if err := importKeyProviders(options.Keys, server.secrets, server.keys); err != nil {
		return nil, fmt.Errorf("key config: %w", err)
	}

	dsn, err := getPostgresConnectionString(options, server.secrets)
	if err != nil {
		return nil, fmt.Errorf("postgres dsn: %w", err)
	}
	options.DB.DSN = dsn

	dbKeyProvider, ok := server.keys[options.DBEncryptionKeyProvider]
	if !ok {
		return nil, fmt.Errorf("key provider %s not configured", options.DBEncryptionKeyProvider)
	}
	options.DB.EncryptionKeyProvider = dbKeyProvider
	options.DB.RootKeyID = options.DBEncryptionKey

	db, err := data.NewDB(options.DB)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	server.db = db
	server.metricsRegistry = setupMetrics(server.db)

	redisPassword, err := secrets.GetSecret(options.Redis.Password, server.secrets)
	if err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}

	options.Redis.Password = redisPassword
	redis, err := redis.NewRedis(options.Redis)
	if err != nil {
		return nil, err
	}
	server.redis = redis

	if options.EnableTelemetry {
		server.tel = NewTelemetry(server.db, db.DefaultOrgSettings.ID)
	}

	if err := server.loadConfig(server.options.Config); err != nil {
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
func (s *Server) DB() data.GormTxn {
	return s.db
}

func (s *Server) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

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
	if opts.ProxyURL.Host != "" {
		remote := opts.ProxyURL.Value()
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}
		proxy.ErrorLog = log.New(logging.NewFilteredHTTPLogger(), "", 0)

		router.Use(func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		})
		return
	}
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

	tlsConfig, err := tlsConfigFromOptions(s.secrets, s.options.TLS)
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
func getPostgresConnectionString(options Options, secretStorage map[string]secrets.SecretStorage) (string, error) {
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
		pass, err := secrets.GetSecret(options.DBPassword, secretStorage)
		if err != nil {
			return "", fmt.Errorf("postgres secret: %w", err)
		}

		fmt.Fprintf(&pgConn, "password=%s ", pass)
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
