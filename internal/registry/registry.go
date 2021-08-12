//go:generate npm run export --silent --prefix ./ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "./ui/out/" ./ui/out/...

package registry

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/goware/urlx"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	v1 "github.com/infrahq/infra/internal/v1"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

type Options struct {
	DBPath        string
	TLSCache      string
	DefaultApiKey string
	ConfigPath    string
	UIProxy       string
}

func mixedHandlerFunc(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}
		httpHandler.ServeHTTP(w, r)
	})
}

func authMetadata(ctx context.Context, req *http.Request) metadata.MD {
	authorization := req.Header.Get("authorization")
	if authorization != "" {
		return nil
	}

	cookie, err := req.Cookie(CookieTokenName)
	if err != nil {
		return nil
	}

	return metadata.Pairs("authorization", "Bearer "+cookie.Value)
}

func authFilter(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return errors.New("failed to extract metadata from context")
	}

	tokens := md.HeaderMD.Get("gateway-set-auth-token")
	if len(tokens) > 0 {
		setAuthCookie(w, tokens[0])
		return nil
	}

	delTokens := md.HeaderMD.Get("gateway-delete-auth-token")
	if len(delTokens) > 0 {
		deleteAuthCookie(w)
		return nil
	}

	return nil
}

func setAuthCookie(w http.ResponseWriter, token string) {
	expires := time.Now().Add(SessionDuration)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieTokenName,
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "1",
		Expires: expires,
		Path:    "/",
	})
}

func deleteAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieTokenName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
	})
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

	// Load configuration from file
	var contents []byte
	if options.ConfigPath != "" {
		contents, err = ioutil.ReadFile(options.ConfigPath)
		if err != nil {
			zapLogger.Error(err.Error())
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
			return errors.New("invalid initial api key length, the key must be 24 characters")

		}
		apiKey.Key = options.DefaultApiKey
		err := db.Save(&apiKey).Error
		if err != nil {
			return err
		}
	}

	k8s, err := kubernetes.NewKubernetes()
	if err != nil {
		return err
	}

	okta := NewOkta()

	timer := timer.Timer{}
	timer.Start(10, func() {
		var sources []Source

		if err := db.Find(&sources).Error; err != nil {
			zapLogger.Error(err.Error())
		}

		for _, s := range sources {
			err = s.SyncUsers(db, k8s, okta)
			if err != nil {
				zapLogger.Error(err.Error())
			}
		}
	})

	defer timer.Stop()

	if err := os.MkdirAll(options.TLSCache, os.ModePerm); err != nil {
		return err
	}

	server := &V1Server{
		db:   db,
		okta: okta,
		k8s:  k8s,
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_zap.UnaryServerInterceptor(zapLogger),
			authInterceptor(db),
		)),
	)
	v1.RegisterV1Server(grpcServer, server)
	reflection.Register(grpcServer)

	gwmux := runtime.NewServeMux(runtime.WithMetadata(authMetadata), runtime.WithForwardResponseOption(authFilter))
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	err = v1.RegisterV1HandlerFromEndpoint(context.Background(), gwmux, "localhost:443", []grpc.DialOption{grpc.WithTransportCredentials(creds)})
	if err != nil {
		return err
	}

	httpHandlers := &Http{
		db:     db,
		logger: zapLogger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", httpHandlers.Healthz)
	mux.HandleFunc("/.well-known/jwks.json", httpHandlers.WellKnownJWKs)
	mux.Handle("/v1/", gwmux)

	if options.UIProxy != "" {
		remote, err := urlx.Parse(options.UIProxy)
		if err != nil {
			return err
		}
		mux.Handle("/", httpHandlers.loginRedirectMiddleware(httputil.NewSingleHostReverseProxy(remote)))
	} else {
		mux.Handle("/", httpHandlers.loginRedirectMiddleware(gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}}))))
	}

	mixedHandler := mixedHandlerFunc(grpcServer, ZapLoggerHttpMiddleware(zapLogger, mux))
	plaintextServer := http.Server{
		Addr:    ":80",
		Handler: h2c.NewHandler(mixedHandler, &http2.Server{}),
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
		Handler:   h2c.NewHandler(mixedHandler, &http2.Server{}),
	}

	return tlsServer.ListenAndServeTLS("", "")
}
