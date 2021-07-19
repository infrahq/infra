//go:generate npm run export --silent --prefix ../ui
//go:generate go-bindata -pkg registry -nocompress -o ./bindata_ui.go -prefix "../ui/out/" ../ui/out/...

package registry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/goware/urlx"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	timer "github.com/infrahq/infra/internal/timer"
	v1 "github.com/infrahq/infra/internal/v1"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	CookieTokenName = "token"
	CookieLoginName = "login"
)

type Options struct {
	DBPath        string
	TLSCache      string
	DefaultApiKey string
	ConfigPath    string
	UI            bool
	UIProxy       string
}

func combinedHandlerFunc(grpcServer *grpc.Server, httpHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
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

func setAuthCookie(w http.ResponseWriter, token string) {
	expires := time.Now().Add(SessionDuration)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieTokenName,
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "1",
		Expires: expires,
		Path:    "/",
		Secure:  true,
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
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    CookieLoginName,
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
		Secure:  true,
	})
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

		for _, s := range sources {
			err = s.SyncUsers(db, okta)
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

	httpHandlers := &Http{
		db: db,
	}

	gwmux := runtime.NewServeMux(runtime.WithMetadata(authMetadata), runtime.WithForwardResponseOption(authFilter))
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	err = v1.RegisterV1HandlerFromEndpoint(context.Background(), gwmux, "localhost:443", []grpc.DialOption{grpc.WithTransportCredentials(creds)})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", httpHandlers.Healthz)
	mux.HandleFunc("/.well-known/jwks.json", httpHandlers.WellKnownJWKs)
	mux.Handle("/v1/", gwmux)

	loginRedirectMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ext := filepath.Ext(r.URL.Path)
			if ext != "" && ext != ".html" {
				next.ServeHTTP(w, r)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/_next") {
				next.ServeHTTP(w, r)
				return
			}

			token, err := r.Cookie(CookieTokenName)
			if err != nil && !errors.Is(err, http.ErrNoCookie) {
				fmt.Println(err)
				return
			}

			if errors.Is(err, http.ErrNoCookie) {
				res, err := server.Status(context.Background(), &emptypb.Empty{})
				if err != nil {
					fmt.Println(err)
					return
				}

				if !res.Admin && !strings.HasPrefix(r.URL.Path, "/signup") {
					http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
					return
				} else if res.Admin && !strings.HasPrefix(r.URL.Path, "/login") {
					params := url.Values{}
					path := "/login"

					next := ""
					if r.URL.Path != "/" {
						next += r.URL.Path
					}
					if r.URL.RawQuery != "" {
						next += "?" + r.URL.RawQuery
					}

					if next != "" {
						params.Add("next", next)
						path = "/login?" + params.Encode()
					}

					http.Redirect(w, r, path, http.StatusTemporaryRedirect)
					return
				}
			} else if strings.HasPrefix(r.URL.Path, "/login") || strings.HasPrefix(r.URL.Path, "/signup") {
				keys, ok := r.URL.Query()["next"]
				if !ok || len(keys[0]) < 1 {
					http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
					return
				} else {
					http.Redirect(w, r, keys[0], http.StatusTemporaryRedirect)
					return
				}
			}

			if token != nil {
				_, err = ValidateAndGetToken(db, token.Value)
				if err != nil {
					deleteAuthCookie(w)
					http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				}
			}

			next.ServeHTTP(w, r)
		})
	}

	if options.UIProxy != "" {
		remote, err := urlx.Parse(options.UIProxy)
		if err != nil {
			return err
		}
		mux.Handle("/", loginRedirectMiddleware(httputil.NewSingleHostReverseProxy(remote)))
	} else if options.UI {
		mux.Handle("/", loginRedirectMiddleware(gziphandler.GzipHandler(http.FileServer(&StaticFileSystem{base: &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}}))))
	}

	httpServer := http.Server{
		Addr:    ":80",
		Handler: combinedHandlerFunc(grpcServer, mux),
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			zapLogger.Error(err.Error())
		}
	}()

	tlsServer := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   combinedHandlerFunc(grpcServer, mux),
	}

	return tlsServer.ListenAndServeTLS("", "")
}
