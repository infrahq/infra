//go:generate npm run export --silent --prefix ./ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "./ui/out/" ./ui/out/...

package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"gorm.io/gorm"
)

type Options struct {
	DBPath               string
	TLSCache             string
	RootAPIKey           string
	EngineAPIKey         string
	ConfigPath           string
	UI                   bool
	UIProxy              string
	SyncInterval         int
	EnableTelemetry      bool
	EnableCrashReporting bool
}

type Registry struct {
	options  Options
	db       *gorm.DB
	logger   *zap.Logger
	settings Settings
	k8s      *kubernetes.Kubernetes
	okta     Okta
	tel      *Telemetry
}

const (
	rootAPIKeyName   = "root"
	engineAPIKeyName = "engine"
)

// syncSources polls every known source for users and groups
func (r *Registry) syncSources() {
	var sources []Source
	if err := r.db.Find(&sources).Error; err != nil {
		r.logger.Sugar().Errorf("could not find sync sources: %w", err)
	}

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			r.logger.Sugar().Debug("synchronizing okta source")

			err := s.SyncUsers(r)
			if err != nil {
				r.logger.Sugar().Errorf("sync okta users: %w", err)
			}

			err = s.SyncGroups(r)
			if err != nil {
				r.logger.Sugar().Errorf("sync okta groups: %w", err)
			}
		default:
			r.logger.Sugar().Errorf("skipped validating unknown source kind %s", s.Kind)
		}
	}
}

func Run(options Options) error {
	var err error
	if options.EnableCrashReporting && internal.CrashReportingDSN != "" {
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
			return err
		}

		defer recoverWithSentryHub(sentry.CurrentHub())
	}

	r := &Registry{}

	r.db, err = NewDB(options.DBPath)
	if err != nil {
		return err
	}

	err = r.db.First(&r.settings).Error
	if err != nil {
		return err
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("registryId", r.settings.Id)
	})

	r.logger, err = logging.Build()
	if err != nil {
		return err
	}

	r.k8s, err = kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	if err = r.loadConfigFromFile(); err != nil {
		return fmt.Errorf("loading config from file: %w", err)
	}

	r.okta = NewOkta()

	r.validateSources()
	r.scheduleSyncJobs()

	if err := r.configureTelemetry(); err != nil {
		return err
	}

	if err := r.saveAPIKeys(); err != nil {
		return err
	}

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	if err := r.runServer(); err != nil {
		return err
	}

	return r.logger.Sync()
}

func (r *Registry) loadConfigFromFile() (err error) {
	var contents []byte

	if r.options.ConfigPath != "" {
		contents, err = ioutil.ReadFile(r.options.ConfigPath)
		if err != nil {
			var perr *fs.PathError

			switch {
			case errors.As(err, &perr):
				r.logger.Warn("no config file found at " + r.options.ConfigPath)
			default:
				r.logger.Error(err.Error())
			}
		}
	}

	if len(contents) > 0 {
		err = ImportConfig(r.db, contents)
		if err != nil {
			return err
		}
	} else {
		r.logger.Warn("skipped importing empty config")
	}

	return nil
}

// validateSources validates any existing or imported sources
func (r *Registry) validateSources() {
	var sources []Source
	if err := r.db.Find(&sources).Error; err != nil {
		r.logger.Sugar().Error("find sources to validate: %w", err)
	}

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			if err := s.Validate(r.db, r.k8s, r.okta); err != nil {
				r.logger.Sugar().Errorf("could not validate okta: %w", err)
			}
		default:
			r.logger.Sugar().Errorf("skipped validating unknown source kind %s", s.Kind)
		}
	}
}

// schedule the user and group sync jobs
func (r *Registry) scheduleSyncJobs() {
	interval := 60 * time.Second
	if r.options.SyncInterval > 0 {
		interval = time.Duration(r.options.SyncInterval) * time.Second
	} else {
		envSync := os.Getenv("INFRA_SYNC_INTERVAL_SECONDS")
		if envSync != "" {
			envInterval, err := strconv.Atoi(envSync)
			if err != nil {
				r.logger.Error("invalid INFRA_SYNC_INTERVAL_SECONDS env: " + err.Error())
			} else {
				interval = time.Duration(envInterval) * time.Second
			}
		}
	}

	// be careful with this sync job, there are Okta rate limits on these requests
	syncSourcesTimer := timer.NewTimer()
	defer syncSourcesTimer.Stop()
	syncSourcesTimer.Start(interval, func() {
		hub := newSentryHub("sync_sources_timer")
		defer recoverWithSentryHub(hub)

		r.syncSources()
	})

	// schedule destination sync job
	syncDestinationsTimer := timer.NewTimer()
	defer syncDestinationsTimer.Stop()
	syncDestinationsTimer.Start(5*time.Minute, func() {
		hub := newSentryHub("sync_destinations_timer")
		defer recoverWithSentryHub(hub)

		now := time.Now()

		var destinations []Destination
		if err := r.db.Find(&destinations).Error; err != nil {
			r.logger.Error(err.Error())
		}

		for i, d := range destinations {
			expiry := time.Unix(d.Updated, 0).Add(time.Hour * 1)
			if expiry.Before(now) {
				if err := r.db.Delete(&destinations[i]).Error; err != nil {
					r.logger.Error(err.Error())
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
			logging.L.Sugar().Debug(err)
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
	} else if r.options.UI {
		mux.Handle("/", h.loginRedirectMiddleware(gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}}))))
	}

	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	plaintextServer := http.Server{
		Addr:    ":80",
		Handler: ZapLoggerHttpMiddleware(sentryHandler.Handle(mux)),
	}

	go func() {
		err := plaintextServer.ListenAndServe()
		if err != nil {
			r.logger.Error(err.Error())
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
		Handler:   ZapLoggerHttpMiddleware(sentryHandler.Handle(mux)),
	}

	if err := tlsServer.ListenAndServeTLS("", ""); err != nil {
		return err
	}

	return nil
}
