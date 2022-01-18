//go:generate npm run export --silent --prefix ../../ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "../../ui/out/" ../../ui/out/...

package registry

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	timer "github.com/infrahq/infra/internal/timer"
	"github.com/infrahq/infra/secrets"
)

type PostgresOptions struct {
	PostgresHost       string `mapstructure:"host"`
	PostgresPort       int    `mapstructure:"port"`
	PostgresDBName     string `mapstructure:"db-name"`
	PostgresUser       string `mapstructure:"user"`
	PostgresPassword   string `mapstructure:"password"`
	PostgresParameters string `mapstructure:"parameters"`
}

type Options struct {
	ConfigPath      string          `mapstructure:"config-path"`
	DBFile          string          `mapstructure:"db-file"`
	TLSCache        string          `mapstructure:"tls-cache"`
	RootAPIToken    string          `mapstructure:"root-api-token"`
	EngineAPIToken  string          `mapstructure:"engine-api-token"`
	PostgresOptions PostgresOptions `mapstructure:"pg"`

	EnableTelemetry      bool `mapstructure:"enable-telemetry"`
	EnableCrashReporting bool `mapstructure:"enable-crash-reporting"`

	ProvidersSyncInterval    time.Duration `mapstructure:"providers-sync-interval"`
	DestinationsSyncInterval time.Duration `mapstructure:"destinations-sync-interval"`

	SessionDuration time.Duration `mapstructure:"session-duration"`

	internal.Options `mapstructure:",squash"`
}

type Registry struct {
	options     Options
	config      Config
	db          *gorm.DB
	tel         *Telemetry
	secrets     map[string]secrets.SecretStorage
	keyProvider map[string]secrets.SymmetricKeyProvider
}

const (
	DefaultProvidersSyncInterval    time.Duration = time.Second * 60
	DefaultDestinationsSyncInterval time.Duration = time.Minute * 5
	DefaultSessionDuration          time.Duration = time.Hour * 12
)

func Run(options Options) (err error) {
	r := &Registry{
		options: options,
	}

	if err, ok := r.configureSentry(); ok {
		defer recoverWithSentryHub(sentry.CurrentHub())
	} else if err != nil {
		return fmt.Errorf("configure sentry: %w", err)
	}

	if err = r.loadSecretsConfigFromFile(); err != nil {
		return fmt.Errorf("loading secrets config from file: %w", err)
	}

	driver, err := r.getDatabaseDriver()
	if err != nil {
		return fmt.Errorf("driver: %w", err)
	}

	r.db, err = data.NewDB(driver)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	settings, err := data.InitializeSettings(r.db)
	if err != nil {
		return fmt.Errorf("settings: %w", err)
	}

	if err = r.loadDBKey(); err != nil {
		return fmt.Errorf("loading database key: %w", err)
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("registryId", settings.ID)
	})

	if err = r.loadConfigFromFile(); err != nil {
		return fmt.Errorf("loading config from file: %w", err)
	}

	r.scheduleSyncJobs()

	if err := r.configureTelemetry(); err != nil {
		return fmt.Errorf("configuring telemetry: %w", err)
	}

	if err := SetupMetrics(r.db); err != nil {
		return fmt.Errorf("configuring metrics: %w", err)
	}

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := r.importAPITokens(); err != nil {
		return fmt.Errorf("importing api tokens: %w", err)
	}

	if err := r.runServer(); err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	return logging.L.Sync()
}

func (r *Registry) readConfig() []byte {
	var contents []byte

	if r.options.ConfigPath != "" {
		var err error

		contents, err = ioutil.ReadFile(r.options.ConfigPath)
		if err != nil {
			var perr *fs.PathError

			switch {
			case errors.As(err, &perr):
				logging.S.Warnf("no config file found at %s", r.options.ConfigPath)
			default:
				logging.L.Error(err.Error())
			}
		}
	}

	return contents
}

// loadSecretsConfigFromFile only loads secret providers, this is needed before reading the whole config to connect to a database
func (r *Registry) loadSecretsConfigFromFile() (err error) {
	contents := r.readConfig()
	if len(contents) > 0 {
		err = r.importSecretsConfig(contents)
		if err != nil {
			return err
		}
	} else {
		logging.L.Warn("skipped importing secret providers empty config")
	}

	return nil
}

func (r *Registry) loadConfigFromFile() (err error) {
	contents := r.readConfig()
	if len(contents) > 0 {
		err = r.importConfig(contents)
		if err != nil {
			return err
		}
	} else {
		logging.L.Warn("skipped importing empty config")
	}

	return nil
}

// schedule the user and group sync jobs, does not schedule when the jobs stop running
func (r *Registry) scheduleSyncJobs() {
	// be careful with this sync job, there are Okta rate limits on these requests
	syncProvidersTimer := timer.NewTimer()
	syncProvidersTimer.Start(r.options.ProvidersSyncInterval, func() {
		syncProviders(r)
	})
}

func (r *Registry) configureTelemetry() error {
	var err error

	r.tel, err = NewTelemetry(r.db)
	if err != nil {
		return err
	}

	r.tel.SetEnabled(r.options.EnableTelemetry)

	telemetryTimer := timer.NewTimer()
	telemetryTimer.Start(60*time.Minute, func() {
		if err := r.tel.EnqueueHeartbeat(); err != nil {
			logging.S.Debug(err)
		}
	})

	return nil
}

func serve(server *http.Server) {
	if err := server.ListenAndServe(); err != nil {
		logging.S.Errorf("server: %w", err)
	}
}

func (r *Registry) runServer() error {
	h := HTTP{db: r.db}
	router := gin.New()

	router.Use(gin.Recovery())
	router.GET("/.well-known/jwks.json", h.WellKnownJWKs)
	router.GET("/healthz", Healthz)

	NewAPIMux(r, router.Group("/v1"))

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

	plaintextServer := &http.Server{
		Addr:     ":80",
		Handler:  handlers.CustomLoggingHandler(io.Discard, sentryHandler.Handle(router), logging.ZapLogFormatter),
		ErrorLog: logging.StandardErrorLog(),
	}

	go serve(plaintextServer)

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(r.options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = certs.SelfSignedOrLetsEncryptCert(manager, func() string {
		return ""
	})

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

// configureSentry returns ok:true when sentry is configured and initialized, or false otherwise. It can be used to know if `defer recoverWithSentryHub(sentry.CurrentHub())` can be called
func (r *Registry) configureSentry() (err error, ok bool) {
	if r.options.EnableCrashReporting && internal.CrashReportingDSN != "" {
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

func (r *Registry) getDatabaseDriver() (gorm.Dialector, error) {
	postgres, err := r.getPostgresConnectionString()
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if postgres != "" {
		return data.NewPostgresDriver(postgres)
	}

	return data.NewSQLiteDriver(r.options.DBFile)
}

// getPostgresConnectionString parses postgres configuration options and returns the connection string
func (r *Registry) getPostgresConnectionString() (string, error) {
	options := r.options.PostgresOptions

	var pgConn strings.Builder

	if options.PostgresHost != "" {
		// config has separate postgres parameters set, combine them into a connection DSN now
		fmt.Fprintf(&pgConn, "host=%s ", options.PostgresHost)

		if options.PostgresUser != "" {
			fmt.Fprintf(&pgConn, "user=%s ", options.PostgresUser)

			if options.PostgresPassword != "" {
				pass, err := r.GetSecret(options.PostgresPassword)
				if err != nil {
					return "", fmt.Errorf("postgres secret: %w", err)
				}

				fmt.Fprintf(&pgConn, "password=%s ", pass)
			}
		}

		if options.PostgresPort > 0 {
			fmt.Fprintf(&pgConn, "port=%d ", options.PostgresPort)
		}

		if options.PostgresDBName != "" {
			fmt.Fprintf(&pgConn, "dbname=%s ", options.PostgresDBName)
		}

		if options.PostgresParameters != "" {
			fmt.Fprintf(&pgConn, "%s", options.PostgresParameters)
		}
	}

	return strings.TrimSpace(pgConn.String()), nil
}

// GetSecret implements the secret definition scheme for Infra.
// eg plaintext:pass123, or kubernetes:infra-okta/apiToken
// it's an abstraction around all secret providers
func (r *Registry) GetSecret(name string) (string, error) {
	var kind string

	if !strings.Contains(name, ":") {
		// we'll have to guess at what type of secret it is.
		// our default guesses are kubernetes, or plain
		if strings.Count(name, "/") == 1 {
			// guess kubernetes for historical reasons
			kind = "kubernetes"
		} else {
			// guess plain because users sometimes mistake the field for plaintext
			kind = "plaintext"
		}

		logging.S.Warnf("Secret kind was not specified, expecting secrets in the format <kind>:<secret name>. Assuming its kind is %q", kind)
	} else {
		parts := strings.SplitN(name, ":", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("unexpected secret provider format %q. Expecting <kind>:<secret name>, eg env:API_TOKEN", name)
		}
		kind = parts[0]
		name = parts[1]
	}

	secretProvider, found := r.secrets[kind]
	if !found {
		return "", fmt.Errorf("secret provider %q not found in configuration for field %q", kind, name)
	}

	b, err := secretProvider.GetSecret(name)
	if err != nil {
		return "", fmt.Errorf("getting secret: %w", err)
	}

	if b == nil {
		return "", nil
	}

	return string(b), nil
}

var dbKeyName = "dbkey"

// load encrypted db key from database
func (r *Registry) loadDBKey() error {
	keyRec, err := data.GetKey(r.db, data.ByName(dbKeyName))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return r.createDBKey()
		}

		return err
	}

	kp := r.keyProvider["default"]

	sKey, err := kp.DecryptDataKey(keyRec.RootKeyID, keyRec.Encrypted)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
}

// creates db key
func (r *Registry) createDBKey() error {
	kp := r.keyProvider["default"]

	sKey, err := kp.GenerateDataKey("")
	if err != nil {
		return err
	}

	key := &models.Key{
		Name:      dbKeyName,
		Encrypted: sKey.Encrypted,
		Algorithm: sKey.Algorithm,
		RootKeyID: sKey.RootKeyID,
	}

	_, err = data.CreateKey(r.db, key)
	if err != nil {
		return err
	}

	models.SymmetricKey = sKey

	return nil
}
