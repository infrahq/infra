//go:generate npm run export --silent --prefix ../../ui
//go:generate go-bindata -pkg server -nocompress -o ./bindata_ui.go -prefix "../../ui/out/" ../../ui/out/...

package server

import (
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/square/go-jose.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/config"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	timer "github.com/infrahq/infra/internal/timer"
	"github.com/infrahq/infra/pki"
	"github.com/infrahq/infra/secrets"
)

type Options struct {
	TLSCache             string        `mapstructure:"tlsCache"`
	AdminAccessKey       string        `mapstructure:"adminAccessKey"`
	AccessKey            string        `mapstructure:"accessKey"`
	EnableTelemetry      bool          `mapstructure:"enableTelemetry"`
	EnableCrashReporting bool          `mapstructure:"enableCrashReporting"`
	EnableUI             bool          `mapstructure:"enableUI"`
	UIProxyURL           string        `mapstructure:"uiProxyURL"`
	EnableSetup          bool          `mapstructure:"enableSetup"`
	SessionDuration      time.Duration `mapstructure:"sessionDuration"`
	InfraHomeDir         string

	DBFile                  string `mapstructure:"dbFile" `
	DBEncryptionKey         string `mapstructure:"dbEncryptionKey"`
	DBEncryptionKeyProvider string `mapstructure:"dbEncryptionKeyProvider"`
	DBHost                  string `mapstructure:"dbHost" `
	DBPort                  int    `mapstructure:"dbPort"`
	DBName                  string `mapstructure:"dbName"`
	DBUser                  string `mapstructure:"dbUser"`
	DBPassword              string `mapstructure:"dbPassword"`
	DBParameters            string `mapstructure:"dbParameters"`

	Keys    []KeyProvider    `mapstructure:"keys"`
	Secrets []SecretProvider `mapstructure:"secrets"`

	Import *config.Config `mapstructure:"import"`

	NetworkEncryption string `yaml:"networkEncryption"` // mtls (default), e2ee, none.
	Certificates      struct {
		TrustInitialClientPublicKey string `yaml:"trustInitialClientPublicKey"`
		InitialRootCACert           string `yaml:"initialRootCACert"`
		InitialRootCAPublicKey      string `yaml:"initialRootCAPublicKey"`
		FullKeyRotationInDays       int    `yaml:"fullKeyRotationInDays"` // 365 default
	} `yaml:"certificates"`
}

type Server struct {
	options             Options
	db                  *gorm.DB
	tel                 *Telemetry
	secrets             map[string]secrets.SecretStorage
	keys                map[string]secrets.SymmetricKeyProvider
	certificateProvider pki.CertificateProvider
}

func Run(options Options) (err error) {
	server := &Server{
		options: options,
	}

	if err := validate.Struct(options); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	if err, ok := server.configureSentry(); ok {
		defer recoverWithSentryHub(sentry.CurrentHub())
	} else if err != nil {
		return fmt.Errorf("configure sentry: %w", err)
	}

	if err := server.importSecrets(); err != nil {
		return fmt.Errorf("secrets config: %w", err)
	}

	if err := server.importSecretKeys(); err != nil {
		return fmt.Errorf("key config: %w", err)
	}

	driver, err := server.getDatabaseDriver()
	if err != nil {
		return fmt.Errorf("driver: %w", err)
	}

	server.db, err = data.NewDB(driver)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	if err = server.loadDBKey(); err != nil {
		return fmt.Errorf("loading database key: %w", err)
	}

	if err = server.loadCertificates(); err != nil {
		return fmt.Errorf("loading certificate provider: %w", err)
	}

	if options.EnableTelemetry {
		if err := configureTelemetry(server.db); err != nil {
			return fmt.Errorf("configuring telemetry: %w", err)
		}
	}

	if err := SetupMetrics(server.db); err != nil {
		return fmt.Errorf("configuring metrics: %w", err)
	}

	if err := os.MkdirAll(server.options.TLSCache, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := server.importAccessKeys(); err != nil {
		return fmt.Errorf("importing access keys: %w", err)
	}

	settings, err := data.InitializeSettings(server.db, server.setupRequired())
	if err != nil {
		return fmt.Errorf("settings: %w", err)
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("serverId", settings.ID)
	})

	if err := server.runServer(); err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	if err := server.importConfig(); err != nil {
		logging.S.Error(fmt.Errorf("import config: %w", err))
	}

	return logging.L.Sync()
}

func configureTelemetry(db *gorm.DB) error {
	tel, err := NewTelemetry(db)
	if err != nil {
		return err
	}

	telemetryTimer := timer.NewTimer()
	telemetryTimer.Start(1*time.Hour, func() {
		if err := tel.EnqueueHeartbeat(); err != nil {
			logging.S.Debug(err)
		}
	})

	return nil
}

func (s *Server) loadCertificates() (err error) {
	if s.options.Certificates.FullKeyRotationInDays == 0 {
		s.options.Certificates.FullKeyRotationInDays = 365
	}

	fullRotationInDays := s.options.Certificates.FullKeyRotationInDays

	keyPath := path.Join(s.options.InfraHomeDir, "keys")
	_ = os.MkdirAll(keyPath, 0o700)

	// TODO: check certificate provider from config
	s.certificateProvider, err = pki.NewNativeCertificateProvider(s.db, pki.NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: fullRotationInDays,
		InitialRootCAPublicKey:        []byte(s.options.Certificates.InitialRootCAPublicKey),
		InitialRootCACert:             []byte(s.options.Certificates.InitialRootCACert),
	})
	if err != nil {
		return err
	}

	// if there's no active CAs, try loading them from options.
	cert := s.options.Certificates.InitialRootCACert
	key := s.options.Certificates.InitialRootCAPublicKey

	if len(s.certificateProvider.ActiveCAs()) == 0 && len(cert) > 0 && len(key) > 0 {
		jsonBytes := fmt.Sprintf(`{"ServerKey":{"CertPEM":"%s", "PublicKey":"%s"}}`, cert, key)
		kp := &pki.KeyPair{}
		err := json.Unmarshal([]byte(jsonBytes), kp)
		if err != nil {
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

	path := filepath.Join(s.options.InfraHomeDir, "keys", "trusted-client-keys")
	err = filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			pems, raw, err := pki.ReadFromPEMFile(path)
			if err != nil {
				return fmt.Errorf("reading pem file %q: %w", info.Name(), err)
			}

			cert, err := x509.ParseCertificate(pems[0].Bytes)
			if err != nil {
				return err
			}

			name := cert.Subject.CommonName
			ident := ""
			switch {
			case strings.HasPrefix(name, "User"):
				ident = "u:" + strings.Split(name, " ")[1]
			case strings.HasPrefix(name, "Machine"):
				ident = "m:" + strings.Split(name, " ")[1]
			}

			tc := &models.TrustedCertificate{
				KeyAlgorithm:     x509.PureEd25519.String(),
				SigningAlgorithm: x509.Ed25519.String(),
				PublicKey:        models.Base64(cert.PublicKey.(ed25519.PublicKey)),
				CertPEM:          raw,
				ExpiresAt:        cert.NotAfter,
				Identity:         ident,
			}

			// insert into db.
			err = data.TrustPublicKey(s.db, tc)
			if err != nil {
				return fmt.Errorf("saving trusted public key: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func serve(server *http.Server) {
	if err := server.ListenAndServe(); err != nil {
		logging.S.Errorf("server: %w", err)
	}
}

func (s *Server) wellKnownJWKsHandler(c *gin.Context) {
	settings, err := data.GetSettings(s.db)
	if err != nil {
		sendAPIError(c, fmt.Errorf("could not get JWKs"))
		return
	}

	var pubKey jose.JSONWebKey
	if err := pubKey.UnmarshalJSON(settings.PublicJWK); err != nil {
		sendAPIError(c, fmt.Errorf("could not get JWKs"))
		return
	}

	c.JSON(http.StatusOK, struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}{
		[]jose.JSONWebKey{pubKey},
	})
}

func (s *Server) healthHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (s *Server) ui(router *gin.Engine) error {
	if s.options.EnableUI {
		if s.options.UIProxyURL != "" {
			remote, err := urlx.Parse(s.options.UIProxyURL)
			if err != nil {
				return err
			}

			proxy := httputil.NewSingleHostReverseProxy(remote)
			proxy.Director = func(req *http.Request) {
				req.Host = remote.Host
				req.URL.Scheme = remote.Scheme
				req.URL.Host = remote.Host
			}

			router.Use(func(c *gin.Context) {
				proxy.ServeHTTP(c.Writer, c.Request)
			})
		} else {
			assetFS := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
			staticFS := &StaticFileSystem{base: assetFS}
			router.Use(gzip.Gzip(gzip.DefaultCompression), static.Serve("/", staticFS))

			// 404
			router.NoRoute(func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.Path, "/v1") {
					c.Status(404)
					c.Writer.WriteHeaderNow()
					return
				}

				c.Status(http.StatusNotFound)
				buf, err := assetFS.Asset("404.html")
				if err != nil {
					logging.S.Error(err)
				}

				_, err = c.Writer.Write(buf)
				if err != nil {
					logging.S.Error(err)
				}

				c.Status(http.StatusNotFound)
			})
		}
	}

	return nil
}

func (s *Server) runServer() error {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())
	router.GET("/.well-known/jwks.json", s.wellKnownJWKsHandler)
	router.GET("/healthz", s.healthHandler)

	NewAPIMux(s, router.Group("/v1"))

	if err := s.ui(router); err != nil {
		return err
	}

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	metrics := gin.New()
	metrics.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	metricsServer := &http.Server{
		Addr:     ":9090",
		Handler:  handlers.CustomLoggingHandler(io.Discard, metrics, logging.ZapLogFormatter),
		ErrorLog: logging.StandardErrorLog(),
	}

	go serve(metricsServer)

	// TODO: Remove plaintext server?
	plaintextServer := &http.Server{
		Addr:     ":80",
		Handler:  handlers.CustomLoggingHandler(io.Discard, sentryHandler.Handle(router), logging.ZapLogFormatter),
		ErrorLog: logging.StandardErrorLog(),
	}

	go serve(plaintextServer)

	if err := os.MkdirAll(s.options.TLSCache, os.ModePerm); err != nil {
		return fmt.Errorf("create tls cache: %w", err)
	}

	tlsConfig, err := s.serverTLSConfig()
	if err != nil {
		return fmt.Errorf("tls config: %w", err)
	}

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   handlers.CustomLoggingHandler(io.Discard, sentryHandler.Handle(router), logging.ZapLogFormatter),
		ErrorLog:  logging.StandardErrorLog(),
	}

	if err := tlsServer.ListenAndServeTLS("", ""); err != nil {
		return err
	}

	return nil
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
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			},
		}, nil
	default: // "none" or blank
		manager := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(s.options.TLSCache),
		}
		tlsConfig := manager.TLSConfig()
		tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, func() string {
			return ""
		})

		return tlsConfig, nil
	}
}

// configureSentry returns ok:true when sentry is configured and initialized, or false otherwise. It can be used to know if `defer recoverWithSentryHub(sentry.CurrentHub())` can be called
func (s *Server) configureSentry() (err error, ok bool) {
	if s.options.EnableCrashReporting && internal.CrashReportingDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              internal.CrashReportingDSN,
			AttachStacktrace: true,
			Release:          internal.Version,
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				event.ServerName = ""
				event.Request = nil
				hint.Request = nil
				return event
			},
		})
		if err != nil {
			return err, false
		}

		return nil, true
	}

	return nil, false
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

func (s *Server) setupRequired() bool {
	if !s.options.EnableSetup {
		return false
	}

	if s.options.AdminAccessKey != "" || s.options.AccessKey != "" {
		return false
	}

	if s.options.Import != nil {
		if len(s.options.Import.Providers) != 0 || len(s.options.Import.Grants) != 0 {
			return false
		}
	}

	machines, err := data.ListMachines(s.db, data.ByName("admin"))
	if err != nil {
		logging.S.Errorf("machines: %w", err)
		return false
	}

	return len(machines) == 0
}
