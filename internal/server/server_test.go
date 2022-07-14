package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
	"time"

	"github.com/infrahq/secrets"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/cmd/types"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
)

func setupServer(t *testing.T, ops ...func(*testing.T, *Options)) *Server {
	t.Helper()
	options := Options{
		SessionDuration:          10 * time.Minute,
		SessionExtensionDeadline: 30 * time.Minute,
	}
	for _, op := range ops {
		op(t, &options)
	}
	s := newServer(options)
	s.db = setupDB(t)

	// TODO: share more of this with Server.New
	err := loadDefaultSecretConfig(s.secrets)
	assert.NilError(t, err)

	err = s.loadConfig(s.options.Config)
	assert.NilError(t, err)

	data.InvalidateCache()
	t.Cleanup(data.InvalidateCache)

	return s
}

func TestGetPostgresConnectionURL(t *testing.T) {
	logging.PatchLogger(t, zerolog.NewTestWriter(t))

	run := func(opts Options) string {
		r := newServer(opts)
		r.secrets = map[string]secrets.SecretStorage{
			"env":       secrets.NewEnvSecretProviderFromConfig(secrets.GenericConfig{}),
			"file":      secrets.NewFileSecretProviderFromConfig(secrets.FileConfig{}),
			"plaintext": secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{}),
		}

		url, err := r.getPostgresConnectionString()
		assert.NilError(t, err)
		return url
	}

	t.Run("empty postgres config", func(t *testing.T) {
		url := run(Options{})
		assert.Assert(t, is.Len(url, 0))
	})

	t.Run("host config", func(t *testing.T) {
		url := run(Options{DBHost: "localhost"})
		assert.Equal(t, url, "host=localhost")
	})

	t.Run("host secret config", func(t *testing.T) {
		t.Setenv("POSTGRES_HOST", "localhost")
		url := run(Options{DBHost: "env:POSTGRES_HOST"})
		assert.Equal(t, url, "host=localhost")
	})

	t.Run("host port config", func(t *testing.T) {
		url := run(Options{DBHost: "localhost", DBPort: 5432})
		assert.Equal(t, url, "host=localhost port=5432")
	})

	t.Run("host username config", func(t *testing.T) {
		url := run(Options{DBHost: "localhost", DBUsername: "postgres"})
		assert.Equal(t, url, "host=localhost user=postgres")
	})

	t.Run("host username secret config", func(t *testing.T) {
		t.Setenv("POSTGRES_USERNAME", "postgres")
		url := run(Options{DBHost: "localhost", DBUsername: "env:POSTGRES_USERNAME"})
		assert.Equal(t, url, "host=localhost user=postgres")
	})

	t.Run("host username password config", func(t *testing.T) {
		url := run(Options{DBHost: "localhost", DBUsername: "postgres", DBPassword: "supersecret"})
		assert.Equal(t, url, "host=localhost user=postgres password=supersecret")
	})

	t.Run("host username password secret config", func(t *testing.T) {
		t.Setenv("POSTGRES_PASSWORD", "supersecret")
		url := run(Options{DBHost: "localhost", DBUsername: "postgres", DBPassword: "env:POSTGRES_PASSWORD"})
		assert.Equal(t, url, "host=localhost user=postgres password=supersecret")
	})

	t.Run("host name config", func(t *testing.T) {
		url := run(Options{DBHost: "localhost", DBName: "postgres"})
		assert.Equal(t, url, "host=localhost dbname=postgres")
	})

	t.Run("host name secret config", func(t *testing.T) {
		t.Setenv("POSTGRES_NAME", "postgres")
		url := run(Options{DBHost: "localhost", DBName: "env:POSTGRES_NAME"})
		assert.Equal(t, url, "host=localhost dbname=postgres")
	})
}

func TestServer_Run(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := t.TempDir()
	opts := Options{
		DBEncryptionKeyProvider: "native",
		DBEncryptionKey:         filepath.Join(dir, "sqlite3.db.key"),
		TLSCache:                filepath.Join(dir, "tlscache"),
		DBFile:                  filepath.Join(dir, "sqlite3.db"),
		TLS: TLSOptions{
			CA:           types.StringOrFile(golden.Get(t, "pki/ca.crt")),
			CAPrivateKey: string(golden.Get(t, "pki/ca.key")),
		},
	}

	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("metrics server started", func(t *testing.T) {
		// perform one API call to populate metrics
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/api/version")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get("http://" + srv.Addrs.Metrics.String() + "/metrics")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)
		// the infra http request metric
		assert.Assert(t, is.Contains(string(body), "# HELP http_request_duration_seconds"))
		// standard go metrics
		assert.Assert(t, is.Contains(string(body), "# HELP go_threads"))
		// standard process metrics
		if runtime.GOOS == "linux" {
			assert.Assert(t, is.Contains(string(body), "# HELP process_open_fds"))
		}
	})

	t.Run("http server started", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/healthz")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("https server started", func(t *testing.T) {
		tr := &http.Transport{}
		tr.TLSClientConfig = &tls.Config{
			// TODO: use the actual certs when that is possible
			//nolint:gosec
			InsecureSkipVerify: true,
		}
		client := &http.Client{Transport: tr}

		url := "https://" + srv.Addrs.HTTPS.String() + "/healthz"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		assert.NilError(t, err)

		resp, err := client.Do(req)
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestServer_Run_UIProxy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	message := `message through the proxy`
	uiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(message))
	}))
	t.Cleanup(uiSrv.Close)

	dir := t.TempDir()
	opts := Options{
		DBEncryptionKeyProvider: "native",
		DBEncryptionKey:         filepath.Join(dir, "sqlite3.db.key"),
		TLSCache:                filepath.Join(dir, "tlscache"),
		DBFile:                  filepath.Join(dir, "sqlite3.db"),
		UI:                      UIOptions{Enabled: true},
		EnableSignup:            true,
		TLS: TLSOptions{
			CA:           types.StringOrFile(golden.Get(t, "pki/ca.crt")),
			CAPrivateKey: string(golden.Get(t, "pki/ca.key")),
		},
	}
	assert.NilError(t, opts.UI.ProxyURL.Set(uiSrv.URL))

	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("requests are proxied", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/any-path")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)
		assert.Equal(t, message, string(body))
	})

	t.Run("api routes are available", func(t *testing.T) {
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/api/signup")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body api.SignupEnabledResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		assert.NilError(t, err)

		assert.Assert(t, body.Enabled)
	})
}

func TestServer_GenerateRoutes_NoRoute(t *testing.T) {
	type testCase struct {
		name     string
		path     string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	fs := fstest.MapFS{
		"ui/static/404.html": {
			Data: []byte("<html>404 - example</html>"),
		},
	}
	s := &Server{options: Options{UI: UIOptions{Enabled: true, FS: fs}}}
	router := s.GenerateRoutes(prometheus.NewRegistry())

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNotFound)
		if tc.expected != nil {
			tc.expected(t, resp)
		}
	}

	testCases := []testCase{
		{
			name: "Using application/json",
			path: "/api/not/found",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Accept", "application/json; charset=utf-8")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				contentType := resp.Header().Get("Content-Type")
				expected := "application/json; charset=utf-8"
				assert.Equal(t, contentType, expected)
			},
		},
		{
			name: "text/html",
			path: "/not/found",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Accept", "text/html")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// response should have an html body
				assert.Assert(t, is.Contains(resp.Body.String(), "404 - example"))

			},
		},
		{
			name: "browser text/html",
			path: "/not/found",
			setup: func(t *testing.T, req *http.Request) {
				browserAccept := []string{"text/html", "application/xhtml+xml",
					"application/xml;q=0.9", "image/webp", "image/apng", "*/*;q=0.8"}
				// this order is used in Safari/Chrome
				for _, ah := range browserAccept {
					req.Header.Add("Accept", ah)
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// response should have an html body
				assert.Assert(t, is.Contains(resp.Body.String(), "404 - example"))

			},
		},
		{
			name: "Other type",
			path: "/not/found",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Accept", "*/*")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// response should be plaintext
				assert.Equal(t, "404 not found", resp.Body.String())
			},
		},
		{
			name: "No header",
			path: "/not/found/again",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				// response should be plaintext
				assert.Equal(t, "404 not found", resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestServer_GenerateRoutes_UI(t *testing.T) {
	type testCase struct {
		name         string
		path         string
		expectedCode int
		expected     func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	fs := fstest.MapFS{
		"ui/static/index.html": {
			Data: []byte("<html>infra home</html>"),
		},
		"ui/static/providers/add/details.html": {
			Data: []byte("<html>mokta provider</html>"),
		},
		"ui/static/icon.svg": {
			Data: []byte("<svg>image</svg>"),
		},
	}
	s := &Server{options: Options{UI: UIOptions{Enabled: true, FS: fs}}}
	router := s.GenerateRoutes(prometheus.NewRegistry())

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Check(t, is.Equal(resp.Code, tc.expectedCode))
		if tc.expected != nil {
			tc.expected(t, resp)
		}
	}

	testCases := []testCase{
		{
			name:         "default index",
			path:         "/",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
		{
			name:         "index page redirects",
			path:         "/index.html",
			expectedCode: http.StatusMovedPermanently,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Location")
				assert.Equal(t, actual, "./")
			},
		},
		{
			name:         "page with a path",
			path:         "/providers/add/details.html",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
		{
			name:         "image",
			path:         "/icon.svg",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "image/svg+xml")
			},
		},
		{
			name:         "page without .html suffix",
			path:         "/providers/add/details",
			expectedCode: http.StatusOK,
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				actual := resp.Header().Get("Content-Type")
				assert.Equal(t, actual, "text/html; charset=utf-8")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestServer_PersistSignupUser(t *testing.T) {
	s := setupServer(t, func(_ *testing.T, opts *Options) {
		opts.EnableSignup = true
		opts.SessionDuration = time.Minute
		opts.SessionExtensionDeadline = time.Minute
	})
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	var buf bytes.Buffer
	email := "admin@email.com"
	passwd := "supersecretpassword"

	// run signup for "admin@email.com"
	signupReq := api.SignupRequest{Name: email, Password: passwd}
	err := json.NewEncoder(&buf).Encode(signupReq)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/signup", &buf)
	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

	signupResp := &api.CreateUserResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), signupResp)
	assert.NilError(t, err)

	// login with "admin@email.com" to get an access key
	loginReq := api.LoginRequest{PasswordCredentials: &api.LoginRequestPasswordCredentials{Name: email, Password: passwd}}
	err = json.NewEncoder(&buf).Encode(loginReq)
	assert.NilError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/api/login", &buf)
	resp = httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

	loginResp := &api.LoginResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), loginResp)
	assert.NilError(t, err)

	checkAuthenticated := func() {
		req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessKey)
		resp = httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)
	}

	// try an authenticated endpoint with the access key
	checkAuthenticated()

	// reload server config
	err = s.loadConfig(s.options.Config)
	assert.NilError(t, err)

	// retry the authenticated endpoint
	checkAuthenticated()
}
