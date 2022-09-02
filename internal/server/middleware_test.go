package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	tpatch "github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *data.DB {
	t.Helper()
	driver := database.PostgresDriver(t, "_server")
	if driver == nil {
		lite, err := data.NewSQLiteDriver("file::memory:")
		assert.NilError(t, err)
		driver = &database.Driver{Dialector: lite}
	}

	tpatch.ModelsSymmetricKey(t)
	db, err := data.NewDB(driver.Dialector, nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, db.Close())
	})

	return db
}

func issueToken(t *testing.T, db data.GormTxn, identityName string, sessionDuration time.Duration) string {
	user := &models.Identity{Name: identityName}

	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	provider := data.InfraProvider(db)

	token := &models.AccessKey{
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().Add(sessionDuration).UTC(),
	}
	body, err := data.CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body
}

func TestRequestTimeoutError(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(100 * time.Millisecond))
	router.GET("/", func(c *gin.Context) {
		time.Sleep(110 * time.Millisecond)

		assert.ErrorIs(t, c.Request.Context().Err(), context.DeadlineExceeded)

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequestTimeoutSuccess(t *testing.T) {
	router := gin.New()
	router.Use(TimeoutMiddleware(60 * time.Second))
	router.GET("/", func(c *gin.Context) {
		assert.NilError(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestDBTimeout(t *testing.T) {
	var ctx context.Context
	var cancel context.CancelFunc

	srv := newServer(Options{})
	srv.db = setupDB(t)

	router := gin.New()
	router.Use(
		func(c *gin.Context) {
			// this is a custom copy of the timeout middleware so I can grab and control the cancel() func. Otherwise the test is too flakey with timing race conditions.
			ctx, cancel = context.WithTimeout(c.Request.Context(), 100*time.Millisecond)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			tx, err := srv.db.Begin(c.Request.Context())
			if err != nil {
				sendAPIError(c, err)
				return
			}
			defer func() {
				_ = tx.Rollback()
			}()

			c.Set(access.RequestContextKey, access.RequestContext{DBTxn: tx})
			c.Next()
		},
	)
	router.GET("/", func(c *gin.Context) {
		rCtx := getRequestContext(c)
		cancel()
		_, err := rCtx.DBTxn.Exec("select 1;")
		assert.Error(t, err, "context canceled")

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequireAccessKey(t *testing.T) {
	type testCase struct {
		setup    func(t *testing.T, db data.GormTxn) *http.Request
		expected func(t *testing.T, authned access.Authenticated, err error)
	}
	cases := map[string]testCase{
		"AccessKeyValid": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, actual access.Authenticated, err error) {
				assert.NilError(t, err)
				assert.Equal(t, actual.User.Name, "existing@infrahq.com")
			},
		},
		"ValidAuthCookie": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*1)

				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     cookieAuthorizationName,
					Value:    authentication,
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     cookiePath,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					HttpOnly: true,
				})

				r.Header.Add("Authorization", " ")
				return r
			},
			expected: func(t *testing.T, actual access.Authenticated, err error) {
				assert.NilError(t, err)
				assert.Equal(t, actual.User.Name, "existing@infrahq.com")
			},
		},
		"ValidSignupCookie": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*1)

				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     cookieSignupName,
					Value:    authentication,
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     cookiePath,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					HttpOnly: true,
				})

				r.Header.Add("Authorization", " ")
				return r
			},
			expected: func(t *testing.T, actual access.Authenticated, err error) {
				assert.NilError(t, err)
				assert.Equal(t, actual.User.Name, "existing@infrahq.com")
			},
		},
		"SignupCookieIsUsedOverAuthCookie": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*1)

				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     cookieSignupName,
					Value:    authentication,
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     cookiePath,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					HttpOnly: true,
				})

				r.AddCookie(&http.Cookie{
					Name:     cookieSignupName,
					Value:    "invalid.access.key",
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     cookiePath,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					HttpOnly: true,
				})

				r.Header.Add("Authorization", " ")
				return r
			},
			expected: func(t *testing.T, actual access.Authenticated, err error) {
				assert.NilError(t, err)
				assert.Equal(t, actual.User.Name, "existing@infrahq.com")
			},
		},
		"AccessKeyExpired": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*-1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorIs(t, err, data.ErrAccessKeyExpired)
			},
		},
		"AccessKeyInvalidKey": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				token := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				secret := token[:models.AccessKeySecretLength]
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), secret)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		"AccessKeyNoMatch": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		"AccessKeyInvalidSecret": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				token := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				authentication := fmt.Sprintf("%s.%s", strings.Split(token, ".")[0], generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "access key invalid secret")
			},
		},
		"UnknownAuthenticationMethod": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				authentication, err := generate.CryptoRandom(32, generate.CharsetAlphaNumeric)
				assert.NilError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "invalid access key format")
			},
		},
		"NoAuthentication": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				// nil pointer if we don't seup the request header here
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
			},
		},
		"EmptyAuthentication": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "")
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
			},
		},
		"EmptySpaceAuthentication": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", " ")
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
			},
		},
		"EmptyCookieAuthentication": {
			setup: func(t *testing.T, db data.GormTxn) *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     cookieAuthorizationName,
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     cookiePath,
					SameSite: http.SameSiteStrictMode,
					Secure:   true,
					HttpOnly: true,
				})

				r.Header.Add("Authorization", " ")
				return r
			},
			expected: func(t *testing.T, _ access.Authenticated, err error) {
				assert.ErrorContains(t, err, "skipped validating empty token")
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			db := setupDB(t)

			srv := &Server{
				options: Options{
					BaseDomain: "example.com",
				},
			}

			req := tc.setup(t, db)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			tx := txnForTestCase(t, db)
			authned, err := requireAccessKey(c, tx, srv)
			tc.expected(t, authned, err)
		})
	}
}

func TestHandleInfraDestinationHeader(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()
	db := srv.DB()

	connector := models.Identity{Name: "connectorA"}
	err := data.CreateIdentity(db, &connector)
	assert.NilError(t, err)

	grant := models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(connector.ID),
		Privilege: models.InfraConnectorRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(db, &grant)
	assert.NilError(t, err)

	token := models.AccessKey{
		IssuedFor:  connector.ID,
		ProviderID: data.InfraProvider(db).ID,
		ExpiresAt:  time.Now().Add(time.Hour).UTC(),
	}
	secret, err := data.CreateAccessKey(db, &token)
	assert.NilError(t, err)

	t.Run("good", func(t *testing.T) {
		destination := &models.Destination{Name: t.Name(), UniqueID: t.Name()}
		err := data.CreateDestination(db, destination)
		assert.NilError(t, err)

		r := httptest.NewRequest("GET", "/api/grants", nil)
		r.Header.Set("Infra-Version", apiVersionLatest)
		r.Header.Set("Infra-Destination", destination.UniqueID)
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)

		destination, err = data.GetDestination(db, data.ByOptionalUniqueID(destination.UniqueID))
		assert.NilError(t, err)
		assert.DeepEqual(t, destination.LastSeenAt, time.Now(), opt.TimeWithThreshold(time.Second))
	})

	t.Run("good no destination header", func(t *testing.T) {
		destination := &models.Destination{Name: t.Name(), UniqueID: t.Name()}
		err := data.CreateDestination(db, destination)
		assert.NilError(t, err)

		r := httptest.NewRequest("GET", "/good", nil)
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", secret))
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)

		destination, err = data.GetDestination(db, data.ByOptionalUniqueID(destination.UniqueID))
		assert.NilError(t, err)
		assert.Equal(t, destination.LastSeenAt.UTC(), time.Time{})
	})

	t.Run("good no destination", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/good", nil)
		r.Header.Add("Infra-Destination", "nonexistent")
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", secret))
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)

		_, err := data.GetDestination(db, data.ByOptionalUniqueID("nonexistent"))
		assert.ErrorIs(t, err, internal.ErrNotFound)
	})
}

func TestAuthenticateRequest(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	org := &models.Organization{
		Name:   "The Umbrella Academy",
		Domain: "umbrella.infrahq.com",
	}
	otherOrg := &models.Organization{
		Name:   "The Factory",
		Domain: "the-factory-xyz8.infrahq.com",
	}
	createOrgs(t, srv.db, otherOrg, org)

	tx, err := srv.db.Begin(context.Background())
	assert.NilError(t, err)
	tx = tx.WithOrgID(org.ID)

	user := &models.Identity{
		Name:               "userone@example.com",
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	createIdentities(t, tx, user)

	token := &models.AccessKey{
		IssuedFor:          user.ID,
		ProviderID:         data.InfraProvider(tx).ID,
		ExpiresAt:          time.Now().Add(10 * time.Second),
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}

	key, err := data.CreateAccessKey(tx, token)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())

	httpSrv := httptest.NewServer(routes)
	t.Cleanup(httpSrv.Close)

	type testCase struct {
		name     string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *http.Response)
	}

	run := func(t *testing.T, tc testCase) {
		// Any authenticated route will do
		routeURL := httpSrv.URL + "/api/users/" + user.ID.String()

		// nolint:noctx
		req, err := http.NewRequest("GET", routeURL, nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		client := httpSrv.Client()
		resp, err := client.Do(req)
		assert.NilError(t, err)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "Org ID from access key",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *http.Response) {
				body, err := io.ReadAll(resp.Body)
				assert.NilError(t, err)

				assert.Equal(t, resp.StatusCode, http.StatusOK, string(body))

				respUser := &api.User{}
				assert.NilError(t, json.Unmarshal(body, respUser))
				assert.Equal(t, respUser.ID, user.ID)
			},
		},
		{
			name: "Missing access key",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = org.Domain
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusUnauthorized)
			},
		},
		{
			name: "Org ID from access key and hostname match",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
				req.Host = org.Domain
			},
			expected: func(t *testing.T, resp *http.Response) {
				body, err := io.ReadAll(resp.Body)
				assert.NilError(t, err)

				assert.Equal(t, resp.StatusCode, http.StatusOK, string(body))

				respUser := &api.User{}
				assert.NilError(t, json.Unmarshal(body, respUser))
				assert.Equal(t, respUser.ID, user.ID)
			},
		},
		{
			name: "Org ID from access key and hostname conflict",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
				req.Host = otherOrg.Domain
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestValidateRequestOrganization(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	srv.options.EnableSignup = true // multi-tenant environment
	routes := srv.GenerateRoutes()

	org := &models.Organization{
		Name:   "The Umbrella Academy",
		Domain: "umbrella.infrahq.com",
	}
	otherOrg := &models.Organization{
		Name:   "The Factory",
		Domain: "the-factory-xyz8.infrahq.com",
	}
	createOrgs(t, srv.db, otherOrg, org)

	tx, err := srv.db.Begin(context.Background())
	assert.NilError(t, err)
	tx = tx.WithOrgID(org.ID)

	provider := &models.Provider{
		Name:               "electric",
		Kind:               models.ProviderKindGoogle,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	assert.NilError(t, data.CreateProvider(tx, provider))

	user := &models.Identity{
		Name:               "userone@example.com",
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	createIdentities(t, tx, user)

	token := &models.AccessKey{
		IssuedFor:          user.ID,
		ProviderID:         data.InfraProvider(tx).ID,
		ExpiresAt:          time.Now().Add(10 * time.Second),
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}

	key, err := data.CreateAccessKey(tx, token)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())

	httpSrv := httptest.NewServer(routes)
	t.Cleanup(httpSrv.Close)

	type testCase struct {
		name     string
		route    string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *http.Response)
	}

	run := func(t *testing.T, tc testCase) {
		// Any unauthenticated route will do
		routeURL := httpSrv.URL + "/api/providers"
		if tc.route != "" {
			routeURL = httpSrv.URL + tc.route
		}

		// nolint:noctx
		req, err := http.NewRequest("GET", routeURL, nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		client := httpSrv.Client()
		resp, err := client.Do(req)
		assert.NilError(t, err)

		tc.expected(t, resp)
	}

	expectSuccess := func(t *testing.T, resp *http.Response) {
		body, err := io.ReadAll(resp.Body)
		assert.NilError(t, err)

		assert.Equal(t, resp.StatusCode, http.StatusOK, string(body))

		respProviders := &api.ListResponse[api.Provider]{}
		assert.NilError(t, json.Unmarshal(body, respProviders))
		assert.Equal(t, len(respProviders.Items), 1)
		assert.Equal(t, respProviders.Items[0].ID, provider.ID)
	}

	testCases := []testCase{
		{
			name: "Org ID from access key",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: expectSuccess,
		},
		{
			name: "Org ID from hostname",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = org.Domain
			},
			expected: expectSuccess,
		},
		{
			name: "Org ID from access key and hostname match",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
				req.Host = org.Domain
			},
			expected: expectSuccess,
		},
		{
			name: "Org ID from access key and hostname conflict",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+key)
				req.Host = otherOrg.Domain
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
			},
		},
		{
			name: "missing org with single-tenancy returns default",
			setup: func(t *testing.T, req *http.Request) {
				srv.options.EnableSignup = false
				t.Cleanup(func() {
					srv.options.EnableSignup = true
				})
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusOK)
			},
		},
		{
			name:  "missing org with multi-tenancy, route ignores org",
			route: "/api/version",
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusOK)
			},
		},
		{
			name: "missing org with multi-tenancy, route returns error",
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusBadRequest)
			},
		},
		{
			name: "missing org with multi-tenancy, route returns fake data",
			setup: func(t *testing.T, req *http.Request) {
				t.Skip("TODO: not yet implemented")
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusOK)
				// TODO: check fake data
			},
		},
		{
			name:  "unknown hostname works like missing org",
			route: "/api/version",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = "http://notadomainweknowabout.org/foo"
			},
			expected: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, resp.StatusCode, http.StatusOK)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
