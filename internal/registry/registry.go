//go:generate npm run export --silent --prefix ./ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "./ui/out/" ./ui/out/...

package registry

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
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
	EngineApiKey         string
	ConfigPath           string
	UI                   bool
	UIProxy              string
	SyncInterval         int
	EnableTelemetry      bool
	EnableCrashReporting bool
}

const (
	rootAPIKeyName   = "root"
	engineApiKeyName = "engine"
)

// syncSources polls every known source for users and groups
func syncSources(db *gorm.DB, k8s *kubernetes.Kubernetes, okta Okta, logger *zap.Logger) {
	var sources []Source
	if err := db.Find(&sources).Error; err != nil {
		logger.Sugar().Errorf("could not find sync sources: %w", err)
	}

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			logger.Sugar().Debug("synchronizing okta source")

			err := s.SyncUsers(db, k8s, okta)
			if err != nil {
				logger.Sugar().Errorf("sync okta users: %w", err)
			}

			err = s.SyncGroups(db, k8s, okta)
			if err != nil {
				logger.Sugar().Errorf("sync okta groups: %w", err)
			}
		default:
			logger.Sugar().Errorf("skipped validating unknown source kind %s", s.Kind)
		}
	}
}

func Run(options Options) error {
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

		defer func() {
			err := recover()
			if err != nil {
				sentry.CurrentHub().Recover(err)
				sentry.Flush(time.Second * 5)
			}
		}()
	}

	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	var settings Settings

	err = db.First(&settings).Error
	if err != nil {
		return err
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext("registryId", settings.Id)
	})

	zapLogger, err := logging.Build()
	if err != nil {
		return err
	}

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	// Load configuration from file
	var contents []byte
	if options.ConfigPath != "" {
		contents, err = ioutil.ReadFile(options.ConfigPath)
		if err != nil {
			var perr *fs.PathError

			switch {
			case errors.As(err, &perr):
				zapLogger.Warn("no config file found at " + options.ConfigPath)
			default:
				zapLogger.Error(err.Error())
			}
		}
	}

	if len(contents) > 0 {
		err = ImportConfig(db, contents)
		if err != nil {
			return err
		}
	} else {
		zapLogger.Warn("skipped importing empty config")
	}

	okta := NewOkta()

	// validate any existing or imported sources
	var sources []Source
	if err := db.Find(&sources).Error; err != nil {
		zapLogger.Sugar().Error("find sources to validate: %w", err)
	}

	for _, s := range sources {
		switch s.Kind {
		case SourceKindOkta:
			if err := s.Validate(db, k8s, okta); err != nil {
				zapLogger.Sugar().Errorf("could not validate okta: %w", err)
			}
		default:
			zapLogger.Sugar().Errorf("skipped validating unknown source kind %s", s.Kind)
		}
	}

	// schedule the user and group sync jobs
	interval := 60 * time.Second
	if options.SyncInterval > 0 {
		interval = time.Duration(options.SyncInterval) * time.Second
	} else {
		envSync := os.Getenv("INFRA_SYNC_INTERVAL_SECONDS")
		if envSync != "" {
			envInterval, err := strconv.Atoi(envSync)
			if err != nil {
				zapLogger.Error("invalid INFRA_SYNC_INTERVAL_SECONDS env: " + err.Error())
			} else {
				interval = time.Duration(envInterval) * time.Second
			}
		}
	}

	// be careful with this sync job, there are Okta rate limits on these requests
	syncSourcesTimer := timer.NewTimer()
	defer syncSourcesTimer.Stop()
	syncSourcesTimer.Start(interval, func() {
		syncSources(db, k8s, okta, zapLogger)
	})

	// schedule destination sync job
	syncDestinationsTimer := timer.NewTimer()
	defer syncDestinationsTimer.Stop()
	syncDestinationsTimer.Start(5*time.Minute, func() {
		now := time.Now()

		var destinations []Destination
		if err := db.Find(&destinations).Error; err != nil {
			zapLogger.Error(err.Error())
		}

		for i, d := range destinations {
			expiry := time.Unix(d.Updated, 0).Add(time.Hour * 1)
			if expiry.Before(now) {
				if err = db.Delete(&destinations[i]).Error; err != nil {
					zapLogger.Error(err.Error())
				}
			}
		}
	})

	t, err := NewTelemetry(db)
	if err != nil {
		return err
	}

	t.SetEnabled(options.EnableTelemetry)

	telemetryTimer := timer.NewTimer()
	telemetryTimer.Start(60*time.Minute, func() {
		if err := t.EnqueueHeartbeat(); err != nil {
			logging.L.Sugar().Debug(err)
		}
	})

	defer telemetryTimer.Stop()

	if options.RootAPIKey != "" {
		if len(options.RootAPIKey) != ApiKeyLen {
			return errors.New("invalid root api key length, the key must be 24 characters")
		}

		var rootAPIKey ApiKey

		err = db.FirstOrCreate(&rootAPIKey, &ApiKey{Name: rootAPIKeyName}).Error
		if err != nil {
			return err
		}

		rootAPIKey.Permissions = string(api.STAR)
		rootAPIKey.Key = options.RootAPIKey

		err := db.Save(&rootAPIKey).Error
		if err != nil {
			return err
		}
	}

	if options.EngineApiKey != "" {
		if len(options.EngineApiKey) != ApiKeyLen {
			return errors.New("invalid engine api key length, the key must be 24 characters")
		}

		var engineApiKey ApiKey

		err = db.FirstOrCreate(&engineApiKey, &ApiKey{Name: engineApiKeyName}).Error
		if err != nil {
			return err
		}

		engineApiKey.Permissions = string(api.DESTINATIONS_CREATE) + " " + string(api.ROLES_READ)
		engineApiKey.Key = options.EngineApiKey

		err := db.Save(&engineApiKey).Error
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	h := Http{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	mux.HandleFunc("/.well-known/jwks.json", h.WellKnownJWKs)
	mux.Handle("/v1/", NewApiMux(db, k8s, okta, t))

	if options.UIProxy != "" {
		remote, err := urlx.Parse(options.UIProxy)
		if err != nil {
			return err
		}

		mux.Handle("/", h.loginRedirectMiddleware(httputil.NewSingleHostReverseProxy(remote)))
	} else if options.UI {
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
			zapLogger.Error(err.Error())
		}
	}()

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
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

	err = tlsServer.ListenAndServeTLS("", "")
	if err != nil {
		return err
	}

	return zapLogger.Sync()
}
