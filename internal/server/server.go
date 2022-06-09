package server

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/infrahq/secrets"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/repeat"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/metrics"
)

type Options struct {
	Version                  float64
	TLSCache                 string
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

	Keys    []KeyProvider
	Secrets []SecretProvider

	Config

	Addr ListenerOptions
	UI   UIOptions
}

type ListenerOptions struct {
	HTTP    string
	HTTPS   string
	Metrics string
}

type UIOptions struct {
	Enabled  bool
	ProxyURL types.URL
	// FS is the filesystem which contains the static files for the UI.
	FS fs.FS `config:"-"`
}

type Server struct {
	options  Options
	db       *gorm.DB
	tel      *Telemetry
	secrets  map[string]secrets.SecretStorage
	keys     map[string]secrets.SymmetricKeyProvider
	Addrs    Addrs
	routines []func() error
}

type Addrs struct {
	HTTP    net.Addr
	HTTPS   net.Addr
	Metrics net.Addr
}

// newServer creates a Server with base dependencies initialized to zero values.
func newServer(options Options) *Server {
	options.UI.FS = uiFS
	return &Server{
		options: options,
		secrets: map[string]secrets.SecretStorage{},
		keys:    map[string]secrets.SymmetricKeyProvider{},
	}
}

// New creates a Server, and initializes it. The returned Server is ready to run.
func New(options Options) (*Server, error) {
	server := newServer(options)

	if err := validate.Struct(options); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

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

	server.db, err = data.NewDB(driver, server.loadDBKey)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	_, err = data.InitializeSettings(server.db)
	if err != nil {
		return nil, fmt.Errorf("settings: %w", err)
	}

	if options.EnableTelemetry {
		if err := configureTelemetry(server); err != nil {
			return nil, fmt.Errorf("configuring telemetry: %w", err)
		}
	}

	if err := server.loadConfig(server.options.Config); err != nil {
		return nil, fmt.Errorf("configs: %w", err)
	}

	if err := server.listen(); err != nil {
		return nil, fmt.Errorf("listening: %w", err)
	}
	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	// nolint: errcheck // if logs won't sync there is no way to report this error
	defer logging.L.Sync()

	if s.tel != nil {
		repeat.Start(context.TODO(), 1*time.Hour, func(context.Context) {
			s.tel.EnqueueHeartbeat()
		})
	}

	group, _ := errgroup.WithContext(ctx)
	for i := range s.routines {
		group.Go(s.routines[i])
	}

	logging.S.Infof("starting infra (%s) - http:%s https:%s metrics:%s",
		internal.FullVersion(), s.Addrs.HTTP, s.Addrs.HTTPS, s.Addrs.Metrics)

	return group.Wait()
}

func configureTelemetry(server *Server) error {
	tel, err := NewTelemetry(server.db)
	if err != nil {
		return err
	}
	server.tel = tel

	return nil
}

//go:embed all:ui/*
var uiFS embed.FS

func registerUIRoutes(router *gin.Engine, opts UIOptions) {
	if !opts.Enabled {
		return
	}

	// Proxy requests to an upstream ui server
	if opts.ProxyURL.Host != "" {
		remote := opts.ProxyURL.Value()
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}

		router.Use(func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		})
		return
	}

	staticFS := &StaticFileSystem{base: http.FS(opts.FS)}
	router.Use(gzip.Gzip(gzip.DefaultCompression), static.Serve("/", staticFS))
}

func (s *Server) listen() error {
	ginutil.SetMode()
	promRegistry := SetupMetrics(s.db)
	router := s.GenerateRoutes(promRegistry)

	metricsServer := &http.Server{
		Addr:     s.options.Addr.Metrics,
		Handler:  metrics.NewHandler(promRegistry),
		ErrorLog: logging.StandardErrorLog(),
	}

	var err error
	s.Addrs.Metrics, err = s.setupServer(metricsServer)
	if err != nil {
		return err
	}

	plaintextServer := &http.Server{
		Addr:     s.options.Addr.HTTP,
		Handler:  router,
		ErrorLog: logging.StandardErrorLog(),
	}
	s.Addrs.HTTP, err = s.setupServer(plaintextServer)
	if err != nil {
		return err
	}

	tlsConfig, err := tlsConfigWithCacheDir(s.options.TLSCache)
	if err != nil {
		return fmt.Errorf("tls config: %w", err)
	}

	tlsServer := &http.Server{
		Addr:      s.options.Addr.HTTPS,
		TLSConfig: tlsConfig,
		Handler:   router,
		ErrorLog:  logging.StandardErrorLog(),
	}
	s.Addrs.HTTPS, err = s.setupServer(tlsServer)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) setupServer(server *http.Server) (net.Addr, error) {
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, err
	}

	s.routines = append(s.routines, func() error {
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
	})
	return l.Addr(), nil
}

func tlsConfigWithCacheDir(tlsCacheDir string) (*tls.Config, error) {
	if err := os.MkdirAll(tlsCacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("create tls cache: %w", err)
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(tlsCacheDir),
	}
	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, "")

	return tlsConfig, nil
}

func (s *Server) getDatabaseDriver() (gorm.Dialector, error) {
	postgres, err := s.getPostgresConnectionString()
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if postgres != "" {
		return data.NewPostgresDriver(postgres)
	}

	return data.NewSQLiteDriver(s.options.DBFile)
}

// getPostgresConnectionString parses postgres configuration options and returns the connection string
func (s *Server) getPostgresConnectionString() (string, error) {
	var pgConn strings.Builder

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
			fmt.Fprintf(&pgConn, "%s", s.options.DBParameters)
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
