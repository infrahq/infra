package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
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
	"github.com/infrahq/infra/internal/testing/database"
)

func setupServer(t *testing.T, ops ...func(*testing.T, *Options)) *Server {
	t.Helper()
	options := Options{
		SessionDuration:          10 * time.Minute,
		SessionInactivityTimeout: 30 * time.Minute,
		API: APIOptions{
			RequestTimeout:         time.Minute,
			BlockingRequestTimeout: 5 * time.Minute,
		},
		GoogleClientID:     "123",
		GoogleClientSecret: "abc",
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

	s.metricsRegistry = prometheus.NewRegistry()
	return s
}

func TestGetPostgresConnectionURL(t *testing.T) {
	logging.PatchLogger(t, zerolog.NewTestWriter(t))

	storage := map[string]secrets.SecretStorage{
		"plaintext": secrets.NewPlainSecretProviderFromConfig(secrets.GenericConfig{}),
	}
	options := Options{}

	url, err := getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Assert(t, is.Len(url, 0))

	options.DBHost = "localhost"
	url, err = getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost", url)

	options.DBPort = 5432
	url, err = getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost port=5432", url)

	options.DBUsername = "user"
	url, err = getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost user=user port=5432", url)

	options.DBPassword = "plaintext:secret"
	url, err = getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost user=user password=secret port=5432", url)

	options.DBName = "postgres"
	url, err = getPostgresConnectionString(options, storage)
	assert.NilError(t, err)
	assert.Equal(t, "host=localhost user=user password=secret port=5432 dbname=postgres", url)

	t.Run("connection string with password from secrets", func(t *testing.T) {
		options := Options{
			DBConnectionString: "host=localhost user=user port=5432",
			DBPassword:         "plaintext:foo",
		}
		dsn, err := getPostgresConnectionString(options, storage)
		assert.NilError(t, err)
		assert.Equal(t, "host=localhost user=user port=5432 password=foo", dsn)
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
		TLS: TLSOptions{
			CA:           types.StringOrFile(golden.Get(t, "pki/ca.crt")),
			CAPrivateKey: string(golden.Get(t, "pki/ca.key")),
		},
		API: APIOptions{RequestTimeout: time.Minute},
	}

	driver := database.PostgresDriver(t, "_server_run")
	opts.DBConnectionString = driver.DSN

	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("metrics server started", func(t *testing.T) {
		// perform one API call to populate metrics
		req, err := http.NewRequest("GET", "http://"+srv.Addrs.HTTP.String()+"/api/version", nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp, err := http.DefaultClient.Do(req)
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// nolint:noctx
		req, err = http.NewRequest("GET", "http://"+srv.Addrs.Metrics.String()+"/metrics", nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp, err = http.DefaultClient.Do(req)
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
		// nolint:noctx
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
		// nolint:noctx
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
		EnableSignup:            true,
		BaseDomain:              "example.com",
		TLS: TLSOptions{
			CA:           types.StringOrFile(golden.Get(t, "pki/ca.crt")),
			CAPrivateKey: string(golden.Get(t, "pki/ca.key")),
		},
		API: APIOptions{RequestTimeout: time.Minute},
	}
	assert.NilError(t, opts.UI.ProxyURL.Set(uiSrv.URL))

	driver := database.PostgresDriver(t, "_server_run")
	opts.DBConnectionString = driver.DSN

	srv, err := New(opts)
	assert.NilError(t, err)

	go func() {
		if err := srv.Run(ctx); err != nil {
			t.Errorf("server errored: %v", err)
		}
	}()

	t.Run("requests are proxied", func(t *testing.T) {
		// nolint:noctx
		resp, err := http.Get("http://" + srv.Addrs.HTTP.String() + "/any-path")
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		assert.NilError(t, err)
		assert.Equal(t, message, string(body))
	})

	t.Run("api routes are available", func(t *testing.T) {
		// nolint:noctx
		req, err := http.NewRequest("GET", "http://"+srv.Addrs.HTTP.String()+"/api/signup", nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp, err := http.DefaultClient.Do(req)
		assert.NilError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestServer_GenerateRoutes_NoRoute(t *testing.T) {
	type testCase struct {
		name     string
		path     string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *http.Response)
	}

	message := `message through the proxy`
	uiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(message))
	}))
	t.Cleanup(uiSrv.Close)

	s := setupServer(t)
	assert.NilError(t, s.options.UI.ProxyURL.Set(uiSrv.URL))
	router := s.GenerateRoutes()

	httpSrv := httptest.NewServer(router)
	t.Cleanup(httpSrv.Close)

	run := func(t *testing.T, tc testCase) {
		u := httpSrv.URL + tc.path
		req, err := http.NewRequest(http.MethodGet, u, nil)
		assert.NilError(t, err)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp, err := httpSrv.Client().Do(req)
		assert.NilError(t, err)

		assert.Equal(t, resp.StatusCode, http.StatusNotFound)
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
			expected: func(t *testing.T, resp *http.Response) {
				contentType := resp.Header.Get("Content-Type")
				expected := "application/json; charset=utf-8"
				assert.Equal(t, contentType, expected)
			},
		},
		{
			name: "Other type",
			path: "/api/not/found",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Accept", "*/*")
			},
			expected: func(t *testing.T, resp *http.Response) {
				body, err := io.ReadAll(resp.Body)
				assert.NilError(t, err)

				// response should be plaintext
				assert.Equal(t, "404 not found", string(body))
			},
		},
		{
			name: "No header",
			path: "/api/not/found/again",
			expected: func(t *testing.T, resp *http.Response) {
				body, err := io.ReadAll(resp.Body)
				assert.NilError(t, err)

				// response should be plaintext
				assert.Equal(t, "404 not found", string(body))
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
		opts.BaseDomain = "example.com"
		opts.SessionDuration = time.Minute
		opts.SessionInactivityTimeout = time.Minute
	})
	routes := s.GenerateRoutes()

	var buf bytes.Buffer
	email := "admin@email.com"
	passwd := "supersecretpassword"

	// run signup for "admin@email.com"
	signupReq := api.SignupRequest{
		User: &api.SignupUser{
			UserName: email,
			Password: passwd,
		},
		OrgName:   "infrahq",
		Subdomain: "myorg1243",
	}
	err := json.NewEncoder(&buf).Encode(signupReq)
	assert.NilError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/signup", &buf)
	req.Header.Set("Infra-Version", apiVersionLatest)
	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

	signupResp := &api.SignupResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), signupResp)
	assert.NilError(t, err)

	// login with "admin@email.com" to get an access key
	loginReq := api.LoginRequest{PasswordCredentials: &api.LoginRequestPasswordCredentials{Name: email, Password: passwd}}
	err = json.NewEncoder(&buf).Encode(loginReq)
	assert.NilError(t, err)

	req = httptest.NewRequest(http.MethodPost, "/api/login", &buf)
	req.Header.Set("Infra-Version", apiVersionLatest)
	req.Host = signupResp.Organization.Domain
	resp = httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

	loginResp := &api.LoginResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), loginResp)
	assert.NilError(t, err)

	checkAuthenticated := func() {
		req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessKey)
		req.Header.Set("Infra-Version", apiVersionLatest)
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
