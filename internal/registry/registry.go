//go:generate npm run export --silent --prefix ./ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "./ui/out/" ./ui/out/...

package registry

import (
	"crypto/tls"
	"errors"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	"golang.org/x/crypto/acme/autocert"
)

type Options struct {
	DBPath        string
	TLSCache      string
	DefaultApiKey string
	ConfigPath    string
	UI            bool
	UIProxy       string
	SyncInterval  int
}

func getSelfSignedOrLetsEncryptCert(certManager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	selfSignCache := make(map[string]*tls.Certificate)

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := certManager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		name := hello.ServerName
		if name == "" {
			name = hello.Conn.LocalAddr().String()
		}

		cert, ok := selfSignCache[name]
		if !ok {
			certBytes, keyBytes, err := generate.SelfSignedCert([]string{name})
			if err != nil {
				return nil, err
			}

			keypair, err := tls.X509KeyPair(certBytes, keyBytes)
			if err != nil {
				return nil, err
			}

			selfSignCache[name] = &keypair
			return &keypair, nil
		}

		return cert, nil
	}
}

func Run(options Options) error {
	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	zapLogger, err := logging.Build()
	defer zapLogger.Sync() // flushes buffer, if any
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
			switch err.(type) {
			case *fs.PathError:
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

	// validate any existing or imported sources
	okta := NewOkta()
	var sources []Source
	if err := db.Find(&sources).Error; err != nil {
		zapLogger.Error(err.Error())
	}

	for _, s := range sources {
		err = s.Validate(db, k8s, okta)
		if err != nil {
			zapLogger.Error(err.Error())
		}
	}

	// schedule the user and group sync jobs
	interval := 30
	if options.SyncInterval > 0 {
		interval = options.SyncInterval
		if err != nil {
			zapLogger.Error("invalid sync interval option specified: " + err.Error())
		}
	} else {
		envSync := os.Getenv("INFRA_SYNC_INTERVAL_SECONDS")
		if envSync != "" {
			interval, err = strconv.Atoi(envSync)
			if err != nil {
				zapLogger.Error("invalid INFRA_SYNC_INTERVAL_SECONDS env: " + err.Error())
			}
		}
	}
	timer := timer.Timer{}
	// be careful with this sync job, there are Okta rate limits on these requests
	timer.Start(interval, func() {
		var sources []Source
		if err := db.Find(&sources).Error; err != nil {
			zapLogger.Error(err.Error())
		}

		for _, s := range sources {
			err = s.SyncUsers(db, k8s, okta)
			if err != nil {
				zapLogger.Error(err.Error())
			}
			err = s.SyncGroups(db, k8s, okta)
			if err != nil {
				zapLogger.Error(err.Error())
			}
		}
	})
	defer timer.Stop()

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		return err
	}

	if options.DefaultApiKey != "" {
		if len(options.DefaultApiKey) != API_KEY_LEN {
			return errors.New("invalid initial api key length, the key must be 24 characters")

		}
		apiKey.Key = options.DefaultApiKey
		err := db.Save(&apiKey).Error
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
	mux.Handle("/v1/", NewApiMux(db, k8s, okta))

	if options.UIProxy != "" {
		remote, err := urlx.Parse(options.UIProxy)
		if err != nil {
			return err
		}
		mux.Handle("/", h.loginRedirectMiddleware(httputil.NewSingleHostReverseProxy(remote)))
	} else if options.UI {
		mux.Handle("/", h.loginRedirectMiddleware(gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}}))))
	}

	plaintextServer := http.Server{
		Addr:    ":80",
		Handler: ZapLoggerHttpMiddleware(mux),
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
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(manager)

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   ZapLoggerHttpMiddleware(mux),
	}

	return tlsServer.ListenAndServeTLS("", "")
}
