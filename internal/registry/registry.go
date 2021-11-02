//go:generate npm run export --silent --prefix ./ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "./ui/out/" ./ui/out/...

package registry

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/handlers"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	"github.com/infrahq/infra/secrets"
	"golang.org/x/crypto/acme/autocert"
	"gorm.io/gorm"
)

type Options struct {
	ConfigPath   string `mapstructure:"config-path"`
	DBFile       string `mapstructure:"db-file"`
	TLSCache     string `mapstructure:"tls-cache"`
	RootAPIKey   string `mapstructure:"root-api-key"`
	EngineAPIKey string `mapstructure:"engine-api-key"`

	EnableUI bool   `mapstructure:"enable-ui"`
	UIProxy  string `mapstructure:"ui-proxy"`

	EnableTelemetry      bool `mapstructure:"enable-telemetry"`
	EnableCrashReporting bool `mapstructure:"enable-crash-reporting"`

	ProvidersSyncInterval    time.Duration `mapstructure:"providers-sync-interval"`
	DestinationsSyncInterval time.Duration `mapstructure:"destinations-sync-interval"`

	internal.Options `mapstructure:",squash"`
}

type Registry struct {
	options  Options
	db       *gorm.DB
	settings Settings
	okta     Okta
	tel      *Telemetry
	secrets  map[string]secrets.SecretStorage
}

const (
	rootAPIKeyName                  string        = "root"
	engineAPIKeyName                string        = "engine"
	DefaultProvidersSyncInterval    time.Duration = time.Second * 60
	DefaultDestinationsSyncInterval time.Duration = time.Minute * 5
)

// syncProviders polls every known provider for users and groups
func (r *Registry) syncProviders() {
	var providers []Provider
	if err := r.db.Find(&providers).Error; err != nil {
		logging.S.Errorf("could not find sync providers: %w", err)
	}

	for _, p := range providers {
		switch p.Kind {
		case ProviderKindOkta:
			logging.L.Debug("synchronizing okta provider")

			err := p.SyncUsers(r)
			if err != nil {
				logging.S.Errorf("sync okta users: %w", err)
			}

			err = p.SyncGroups(r)
			if err != nil {
				logging.S.Errorf("sync okta groups: %w", err)
			}
		default:
			logging.S.Errorf("skipped validating unknown provider kind %s", p.Kind)
		}
	}
}

func Run(options Options) (err error) {
	r := &Registry{
		options: options,
	}

	if err, ok := r.configureSentry(); ok {
		defer recoverWithSentryHub(sentry.CurrentHub())
	} else if err != nil {
		return fmt.Errorf("configure sentry: %w", err)
	}

	r.db, err = NewDB(options.DBFile)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	err = r.db.First(&r.settings).Error
	if err != nil {
		return fmt.Errorf("checking db for settings: %w", err)
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("registryId", r.settings.Id)
	})

	if err = r.loadConfigFromFile(); err != nil {
		return fmt.Errorf("loading config from file: %w", err)
	}

	r.okta = NewOkta()

	r.validateProviders()
	r.scheduleSyncJobs()

	if err := r.configureTelemetry(); err != nil {
		return fmt.Errorf("configuring telemetry: %w", err)
	}

	if err := r.saveAPIKeys(); err != nil {
		return fmt.Errorf("saving api keys: %w", err)
	}

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := r.runServer(); err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	return logging.L.Sync()
}

func (r *Registry) loadConfigFromFile() (err error) {
	var contents []byte
	if r.options.ConfigPath != "" {
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

// validateProviders validates any existing or imported providers
func (r *Registry) validateProviders() {
	var providers []Provider
	if err := r.db.Find(&providers).Error; err != nil {
		logging.S.Error("find providers to validate: %w", err)
	}

	for _, p := range providers {
		switch p.Kind {
		case ProviderKindOkta:
			if err := p.Validate(r); err != nil {
				logging.S.Errorf("could not validate okta: %w", err)
			}
		default:
			logging.S.Errorf("skipped validating unknown provider kind %s", p.Kind)
		}
	}
}

// schedule the user and group sync jobs, does not schedule when the jobs stop running
func (r *Registry) scheduleSyncJobs() {
	// be careful with this sync job, there are Okta rate limits on these requests
	syncProvidersTimer := timer.NewTimer()
	syncProvidersTimer.Start(r.options.ProvidersSyncInterval, func() {
		hub := newSentryHub("sync_providers_timer")
		defer recoverWithSentryHub(hub)

		r.syncProviders()
	})

	// schedule destination sync job
	syncDestinationsTimer := timer.NewTimer()
	syncDestinationsTimer.Start(r.options.DestinationsSyncInterval, func() {
		hub := newSentryHub("sync_destinations_timer")
		defer recoverWithSentryHub(hub)

		now := time.Now()

		var destinations []Destination
		if err := r.db.Find(&destinations).Error; err != nil {
			logging.L.Error(err.Error())
		}

		for i, d := range destinations {
			expiry := time.Unix(d.Updated, 0).Add(time.Hour * 1)
			if expiry.Before(now) {
				if err := r.db.Delete(&destinations[i]).Error; err != nil {
					logging.L.Error(err.Error())
				}
			}
		}
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

func (r *Registry) saveAPIKeys() error {
	var rootAPIKey APIKey
	if err := r.db.FirstOrCreate(&rootAPIKey, &APIKey{Name: rootAPIKeyName}).Error; err != nil {
		return err
	}

	rootAPIKeyURI, err := url.Parse(r.options.RootAPIKey)
	if err != nil {
		return err
	}

	switch rootAPIKeyURI.Scheme {
	case "":
		// option does not have a scheme, assume it is plaintext
		rootAPIKey.Key = string(r.options.RootAPIKey)
	case "file":
		// option is a file path, read contents from the path
		contents, err := ioutil.ReadFile(rootAPIKeyURI.Path)
		if err != nil {
			return err
		}

		if len(contents) != APIKeyLen {
			return fmt.Errorf("invalid api key length, the key must be 24 characters")
		}

		rootAPIKey.Key = string(contents)

	default:
		return fmt.Errorf("unsupported secret format %s", rootAPIKeyURI.Scheme)
	}

	rootAPIKey.Permissions = string(api.STAR)

	if err := r.db.Save(&rootAPIKey).Error; err != nil {
		return err
	}

	var engineAPIKey APIKey
	if err := r.db.FirstOrCreate(&engineAPIKey, &APIKey{Name: engineAPIKeyName}).Error; err != nil {
		return err
	}

	engineAPIKeyURI, err := url.Parse(r.options.EngineAPIKey)
	if err != nil {
		return err
	}

	switch engineAPIKeyURI.Scheme {
	case "":
		// option does not have a scheme, assume it is plaintext
		engineAPIKey.Key = string(r.options.EngineAPIKey)
	case "file":
		// option is a file path, read contents from the path
		contents, err := ioutil.ReadFile(engineAPIKeyURI.Path)
		if err != nil {
			return err
		}

		if len(contents) != APIKeyLen {
			return fmt.Errorf("invalid api key length, the key must be 24 characters")
		}

		engineAPIKey.Key = string(contents)
	default:
		return fmt.Errorf("unsupported secret format %s", engineAPIKeyURI.Scheme)
	}

	engineAPIKey.Permissions = strings.Join([]string{
		string(api.ROLES_READ),
		string(api.DESTINATIONS_CREATE),
	}, " ")

	if err := r.db.Save(&engineAPIKey).Error; err != nil {
		return err
	}

	return nil
}

func (r *Registry) runServer() error {
	h := Http{db: r.db}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	mux.HandleFunc("/.well-known/jwks.json", h.WellKnownJWKs)
	mux.Handle("/v1/", NewAPIMux(r))

	if r.options.UIProxy != "" {
		remote, err := urlx.Parse(r.options.UIProxy)
		if err != nil {
			return err
		}

		mux.Handle("/", h.loginRedirectMiddleware(httputil.NewSingleHostReverseProxy(remote)))
	} else if r.options.EnableUI {
		mux.Handle("/", h.loginRedirectMiddleware(gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}}))))
	}

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	plaintextServer := http.Server{
		Addr:    ":80",
		Handler: handlers.CustomLoggingHandler(io.Discard, sentryHandler.Handle(mux), logging.ZapLogFormatter),
	}

	go func() {
		err := plaintextServer.ListenAndServe()
		if err != nil {
			logging.L.Error(err.Error())
		}
	}()

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
		Handler:   handlers.CustomLoggingHandler(io.Discard, sentryHandler.Handle(mux), logging.ZapLogFormatter),
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
			return "", fmt.Errorf("unexpected secret provider format %q. Expecting <kind>:<secret name>, eg env:API_KEY", name)
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
