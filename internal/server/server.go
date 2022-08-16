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

	"github.com/gin-gonic/gin"
	"github.com/infrahq/secrets"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/metrics"
)

type Options struct {
	Version                  float64
	TLSCache                 string // TODO: move this to TLS.CacheDir
	EnableTelemetry          bool
	EnableSignup             bool
	SessionDuration          time.Duration
	SessionExtensionDeadline time.Duration

	DBFile                  string
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

	Hostname string

	Keys    []KeyProvider
	Secrets []SecretProvider

	Config

	Addr ListenerOptions
	UI   UIOptions
	TLS  TLSOptions
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

type Server struct {
	options Options
	// TODO: consolidate these two
	db     *gorm.DB
	dataDB *data.DB

	tel      *Telemetry
	secrets  map[string]secrets.SecretStorage
	keys     map[string]secrets.SymmetricKeyProvider
	Addrs    Addrs
	routines []routine
}

type Addrs struct {
	HTTP    net.Addr
	HTTPS   net.Addr
	Metrics net.Addr
}

// newServer creates a Server with base dependencies initialized to zero values.
func newServer(options Options) *Server {
	if options.Hostname == "" {
		options.Hostname = "example.com"
	}
	return &Server{
		options: options,
		secrets: map[string]secrets.SecretStorage{},
		keys:    map[string]secrets.SymmetricKeyProvider{},
	}
}

// New creates a Server, and initializes it. The returned Server is ready to run.
func New(options Options) (*Server, error) {
	server := newServer(options)

	if err := importSecrets(options.Secrets, server.secrets); err != nil {
		return nil, fmt.Errorf("secrets config: %w", err)
	}

	if err := importKeyProviders(options.Keys, server.secrets, server.keys); err != nil {
		return nil, fmt.Errorf("key config: %w", err)
	}

	driver, err := server.getDatabaseDriver()
	if err != nil {
		return nil, fmt.Errorf("driver: %w", err)
	}

	db, err := data.NewDB(driver, server.loadDBKey)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	// TODO: store data.DB on server
	server.db = db.DB
	server.dataDB = db

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

func (s *Server) Run(ctx context.Context) error {
	if s.tel != nil {
		repeat.Start(ctx, 1*time.Hour, func(context.Context) {
			s.tel.EnqueueHeartbeat()
		})
	}

	group, _ := errgroup.WithContext(ctx)
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

	if sqlDB, err := s.db.DB(); err != nil {
		logging.L.Warn().Err(err).Msg("failed to get database conn to close")
	} else {
		if err := sqlDB.Close(); err != nil {
			logging.L.Warn().Err(err).Msg("failed to close database connection")
		}
	}

	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
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
	promRegistry := setupMetrics(s.db)
	router := s.GenerateRoutes(promRegistry)

	httpErrorLog := log.New(logging.NewFilteredHTTPLogger(), "", 0)
	metricsServer := &http.Server{
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		Addr:              s.options.Addr.Metrics,
		Handler:           metrics.NewHandler(promRegistry),
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

func (s *Server) getDatabaseDriver() (gorm.Dialector, error) {
	pgDSN, err := s.getPostgresConnectionString()
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if pgDSN != "" {
		return postgres.Open(pgDSN), nil
	}

	return data.NewSQLiteDriver(s.options.DBFile)
}

// getPostgresConnectionString parses postgres configuration options and returns the connection string
func (s *Server) getPostgresConnectionString() (string, error) {
	var pgConn strings.Builder
	pgConn.WriteString(s.options.DBConnectionString)

	if s.options.DBHost != "" {
		// config has separate postgres parameters set, combine them into a connection DSN now
		fmt.Fprintf(&pgConn, "host=%s ", s.options.DBHost)

		if s.options.DBUsername != "" {
			fmt.Fprintf(&pgConn, "user=%s ", s.options.DBUsername)

			if s.options.DBPassword != "" {
				pass, err := secrets.GetSecret(s.options.DBPassword, s.secrets)
				if err != nil {
					return "", fmt.Errorf("postgres secret: %w", err)
				}

				fmt.Fprintf(&pgConn, "password=%s ", pass)
			}
		}

		if s.options.DBPort > 0 {
			fmt.Fprintf(&pgConn, "port=%d ", s.options.DBPort)
		}

		if s.options.DBName != "" {
			fmt.Fprintf(&pgConn, "dbname=%s ", s.options.DBName)
		}

		if s.options.DBParameters != "" {
			fmt.Fprint(&pgConn, s.options.DBParameters)
		}
	}

	return strings.TrimSpace(pgConn.String()), nil
}

var dbKeyName = "dbkey"

// load encrypted db key from database
func (s *Server) loadDBKey(db *gorm.DB) error {
	provider, ok := s.keys[s.options.DBEncryptionKeyProvider]
	if !ok {
		return fmt.Errorf("key provider %s not configured", s.options.DBEncryptionKeyProvider)
	}

	keyRec, err := data.GetEncryptionKey(db, data.ByName(dbKeyName))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return createDBKey(db, provider, s.options.DBEncryptionKey)
		}

		return err
	}

	sKey, err := provider.DecryptDataKey(s.options.DBEncryptionKey, keyRec.Encrypted)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
}

// creates db key
func createDBKey(db *gorm.DB, provider secrets.SymmetricKeyProvider, rootKeyId string) error {
	sKey, err := provider.GenerateDataKey(rootKeyId)
	if err != nil {
		return err
	}

	key := &models.EncryptionKey{
		Name:      dbKeyName,
		Encrypted: sKey.Encrypted,
		Algorithm: sKey.Algorithm,
		RootKeyID: sKey.RootKeyID,
	}

	_, err = data.CreateEncryptionKey(db, key)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
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
}
