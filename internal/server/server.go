//go:generate ./generate-ui.sh

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
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
	"github.com/infrahq/infra/pki"
)

type Options struct {
	TLSCache        string        `mapstructure:"tlsCache"`
	EnableTelemetry bool          `mapstructure:"enableTelemetry"`
	EnableSignup    bool          `mapstructure:"enableSignup"`
	SessionDuration time.Duration `mapstructure:"sessionDuration"`

	DBFile                  string `mapstructure:"dbFile"`
	DBEncryptionKey         string `mapstructure:"dbEncryptionKey"`
	DBEncryptionKeyProvider string `mapstructure:"dbEncryptionKeyProvider"`
	DBHost                  string `mapstructure:"dbHost" `
	DBPort                  int    `mapstructure:"dbPort"`
	DBName                  string `mapstructure:"dbName"`
	DBUser                  string `mapstructure:"dbUsername"`
	DBPassword              string `mapstructure:"dbPassword"`
	DBParameters            string `mapstructure:"dbParameters"`

	Keys    []KeyProvider    `mapstructure:"keys"`
	Secrets []SecretProvider `mapstructure:"secrets"`

	Config `mapstructure:",squash"`

	NetworkEncryption           string `mapstructure:"networkEncryption"` // mtls (default), e2ee, none.
	TrustInitialClientPublicKey string `mapstructure:"trustInitialClientPublicKey"`
	InitialRootCACert           string `mapstructure:"initialRootCACert"`
	InitialRootCAPublicKey      string `mapstructure:"initialRootCAPublicKey"`
	FullKeyRotationInDays       int    `mapstructure:"fullKeyRotationInDays"` // 365 default

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
	ProxyURL types.URL `mapstructure:"proxyURL"`
}

type Server struct {
	options             Options
	db                  *gorm.DB
	tel                 *Telemetry
	secrets             map[string]secrets.SecretStorage
	keys                map[string]secrets.SymmetricKeyProvider
	certificateProvider pki.CertificateProvider
	Addrs               Addrs
	routines            []func() error
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

	server.db, err = data.NewRawDB(driver)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	if err := data.PreMigrate(server.db); err != nil {
		return nil, err
	}

	if err = server.loadDBKey(); err != nil {
		return nil, fmt.Errorf("loading database key: %w", err)
	}

	if err := data.Migrate(server.db); err != nil {
		return nil, err
	}

	if err = server.loadCertificates(); err != nil {
		return nil, fmt.Errorf("loading certificate provider: %w", err)
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

func (s *Server) loadCertificates() (err error) {
	if s.options.FullKeyRotationInDays == 0 {
		s.options.FullKeyRotationInDays = 365
	}

	fullRotationInDays := s.options.FullKeyRotationInDays

	// TODO: check certificate provider from config
	s.certificateProvider, err = pki.NewNativeCertificateProvider(s.db, pki.NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: fullRotationInDays,
		InitialRootCAPublicKey:        []byte(s.options.InitialRootCAPublicKey),
		InitialRootCACert:             []byte(s.options.InitialRootCACert),
	})
	if err != nil {
		return err
	}

	// if there's no active CAs, try loading them from options.
	cert := s.options.InitialRootCACert
	key := s.options.InitialRootCAPublicKey

	if len(s.certificateProvider.ActiveCAs()) == 0 && len(cert) > 0 && len(key) > 0 {
		jsonBytes := fmt.Sprintf(`{"ServerKey":{"CertPEM":"%s", "PublicKey":"%s"}}`, cert, key)

		kp := &pki.KeyPair{}
		if err := json.Unmarshal([]byte(jsonBytes), kp); err != nil {
			return fmt.Errorf("reading initialRootCACert and initialRootCAPublicKey: %w", err)
		}

		err = s.certificateProvider.Preload(kp.CertPEM, kp.PublicKey)
		if err != nil && err.Error() != internal.ErrNotImplemented.Error() {
			return fmt.Errorf("preloading initialRootCACert and initialRootCAPublicKey: %w", err)
		}
	}

	// if still no active CAs, create them
	if len(s.certificateProvider.ActiveCAs()) == 0 {
		logging.S.Info("Creating Root CA certificate")

		if err := s.certificateProvider.CreateCA(); err != nil {
			return fmt.Errorf("creating CA certificates: %w", err)
		}
	}

	// automatically rotate CAs as the oldest one expires
	if len(s.certificateProvider.ActiveCAs()) == 1 {
		logging.S.Info("Rotating Root CA certificate")

		if err := s.certificateProvider.RotateCA(); err != nil {
			return fmt.Errorf("rotating CA: %w", err)
		}
	}

	// if the current cert is going to expire in less than FullKeyRotationDurationInDays/2 days, rotate.
	rotationWindow := time.Now().AddDate(0, 0, fullRotationInDays/2)

	activeCAs := s.certificateProvider.ActiveCAs()
	if len(activeCAs) < 2 || activeCAs[1].NotAfter.Before(rotationWindow) {
		logging.S.Info("Half-Rotating Root CA certificate")

		if err := s.certificateProvider.RotateCA(); err != nil {
			return fmt.Errorf("rotating CA: %w", err)
		}
	}

	if len(s.options.TrustInitialClientPublicKey) > 0 {
		key := s.options.TrustInitialClientPublicKey

		rawKey, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return fmt.Errorf("reading trustInitialClientPublicKey: %w", err)
		}

		tc := &models.TrustedCertificate{
			KeyAlgorithm:     x509.PureEd25519.String(),
			SigningAlgorithm: x509.Ed25519.String(),
			PublicKey:        models.Base64(rawKey),
			// CertPEM:          raw,
			// ExpiresAt:        cert.NotAfter,
			// Identity:         ident,
		}

		err = data.TrustPublicKey(s.db, tc)
		if err != nil {
			return fmt.Errorf("saving trusted public key: %w", err)
		}
	}

	return nil
}

//go:embed all:ui/*
var assetFS embed.FS

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

	staticFS := &StaticFileSystem{base: http.FS(assetFS)}
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

	tlsConfig, err := s.serverTLSConfig()
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

func (s *Server) serverTLSConfig() (*tls.Config, error) {
	switch s.options.NetworkEncryption {
	case "mtls":
		serverTLSCerts, err := s.certificateProvider.TLSCertificates()
		if err != nil {
			return nil, fmt.Errorf("getting tls certs: %w", err)
		}

		caPool := x509.NewCertPool()

		for _, cert := range s.certificateProvider.ActiveCAs() {
			cert := cert
			caPool.AddCert(&cert)
		}

		tcerts, err := data.ListTrustedClientCertificates(s.db)
		if err != nil {
			return nil, err
		}

		for _, tcert := range tcerts {
			p, _ := pem.Decode(tcert.CertPEM)

			cert, err := x509.ParseCertificate(p.Bytes)
			if err != nil {
				return nil, err
			}

			if cert.NotAfter.After(time.Now()) {
				logging.S.Debugf("Trusting user certificate %q\n", cert.Subject.CommonName)
				caPool.AddCert(cert)
			}
		}

		return &tls.Config{
			Certificates: serverTLSCerts,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caPool,
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			},
		}, nil
	default: // "none" or blank
		if err := os.MkdirAll(s.options.TLSCache, 0o700); err != nil {
			return nil, fmt.Errorf("create tls cache: %w", err)
		}

		manager := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(s.options.TLSCache),
		}
		tlsConfig := manager.TLSConfig()
		tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, "")

		return tlsConfig, nil
	}
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

		if s.options.DBUser != "" {
			fmt.Fprintf(&pgConn, "user=%s ", s.options.DBUser)

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
func (s *Server) loadDBKey() error {
	key, ok := s.keys[s.options.DBEncryptionKeyProvider]
	if !ok {
		return fmt.Errorf("key provider %s not configured", s.options.DBEncryptionKeyProvider)
	}

	keyRec, err := data.GetEncryptionKey(s.db, data.ByName(dbKeyName))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return s.createDBKey(key, s.options.DBEncryptionKey)
		}

		return err
	}

	sKey, err := key.DecryptDataKey(s.options.DBEncryptionKey, keyRec.Encrypted)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
}

// creates db key
func (s *Server) createDBKey(provider secrets.SymmetricKeyProvider, rootKeyId string) error {
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

	_, err = data.CreateEncryptionKey(s.db, key)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
}
