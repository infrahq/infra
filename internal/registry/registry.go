package registry

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	v1 "github.com/infrahq/infra/internal/v1"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Options struct {
	DBPath        string
	TLSCache      string
	DefaultApiKey string
	ConfigPath    string
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

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func Run(options Options) error {
	db, err := NewDB(options.DBPath)
	if err != nil {
		return err
	}

	// Load configuration from file
	var contents []byte
	if options.ConfigPath != "" {
		contents, err = ioutil.ReadFile(options.ConfigPath)
		if err != nil {
			fmt.Println("Could not open config file path: ", options.ConfigPath)
		}
	}

	err = ImportConfig(db, contents)
	if err != nil {
		return err
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		return err
	}

	if options.DefaultApiKey != "" {
		if len(options.DefaultApiKey) != API_KEY_LEN {
			return errors.New("invalid initial api key length")

		}
		apiKey.Key = options.DefaultApiKey
		err := db.Save(&apiKey).Error
		if err != nil {
			return err
		}
	}

	okta := NewOkta()

	timer := timer.Timer{}
	timer.Start(10, func() {
		var sources []Source

		if err := db.Find(&sources).Error; err != nil {
			fmt.Println(err)
		}

		for _, p := range sources {
			err = p.SyncUsers(db, okta)
		}
	})

	defer timer.Stop()

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(options.TLSCache),
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = getSelfSignedOrLetsEncryptCert(manager)

	httpHandlers := &Http{
		db: db,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", httpHandlers.Healthz)
	mux.HandleFunc("/.well-known/jwks.json", httpHandlers.WellKnownJWKs)

	server := &V1Server{
		db:   db,
		okta: okta,
	}

	zapLogger, err := logging.Build()
	defer zapLogger.Sync() // flushes buffer, if any
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_zap.UnaryServerInterceptor(zapLogger),
			authInterceptor(db),
		)),
	)
	v1.RegisterV1Server(grpcServer, server)
	reflection.Register(grpcServer)
	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   grpcHandlerFunc(grpcServer, mux),
	}

	return tlsServer.ListenAndServeTLS("", "")
}
