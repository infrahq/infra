package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	return db
}

func issueUserToken(t *testing.T, db *gorm.DB, email, permissions string, sessionDuration time.Duration) string {
	user := &models.User{Email: email, Permissions: permissions}

	err := data.CreateUser(db, user)
	require.NoError(t, err)

	token := &models.AccessKey{
		IssuedFor: user.PolymorphicIdentifier(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	body, err := data.CreateAccessKey(db, token)
	require.NoError(t, err)

	return body
}

func issueMachineToken(t *testing.T, db *gorm.DB, name, permissions string, sessionDuration time.Duration) string {
	machine := &models.Machine{Name: name, Permissions: permissions}

	err := data.CreateMachine(db, machine)
	require.NoError(t, err)

	token := &models.AccessKey{
		IssuedFor: machine.PolymorphicIdentifier(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	body, err := data.CreateAccessKey(db, token)
	require.NoError(t, err)

	return body
}

func TestRequestTimeoutError(t *testing.T) {
	requestTimeout = 100 * time.Millisecond

	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.Use(RequestTimeoutMiddleware())
	router.GET("/", func(c *gin.Context) {
		time.Sleep(110 * time.Millisecond)

		require.Error(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequestTimeoutSuccess(t *testing.T) {
	requestTimeout = 60 * time.Second

	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.Use(RequestTimeoutMiddleware())
	router.GET("/", func(c *gin.Context) {
		require.NoError(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	cases := map[string]map[string]interface{}{
		"AccessKeyValid": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueUserToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, "*", permissions)
			},
		},
		"AccessKeyCookieValid": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueUserToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     CookieAuthorizationName,
					Value:    authentication,
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     CookiePath,
					Domain:   CookieDomain,
					SameSite: http.SameSiteStrictMode,
					Secure:   CookieSecureHTTPSOnly,
					HttpOnly: CookieHTTPOnlyNotJavascriptAccessible,
				})

				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, "*", permissions)
			},
		},
		"AccessKeySetsUserPermissions": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueUserToken(t, db, "existing@infrahq.com", string(access.PermissionTokenCreate), time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(access.PermissionTokenCreate), permissions)
			},
		},
		"AccessKeySetsMachinePermissions": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueMachineToken(t, db, "Wall-E", string(access.PermissionTokenCreate), time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(access.PermissionTokenCreate), permissions)
			},
		},
		"AccessKeyExpired": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueUserToken(t, db, "existing@infrahq.com", "*", time.Minute*-1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "token expired")
			},
		},
		"AccessKeyInvalidKey": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueUserToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				secret := token[:models.AccessKeySecretLength]
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), secret)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "record not found")
			},
		},
		"AccessKeyNoMatch": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), generate.MathRandom(models.AccessKeySecretLength))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "record not found")
			},
		},
		"AccessKeyInvalidSecret": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueUserToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				authentication := fmt.Sprintf("%s.%s", strings.Split(token, ".")[0], generate.MathRandom(models.AccessKeySecretLength))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "access key invalid secret")
			},
		},
		"UnknownAuthenticationMethod": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication, err := generate.CryptoRandom(32)
				require.NoError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "rejected access key format")
			},
		},
		"NoAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				// nil pointer if we don't seup the request header here
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "valid token not found in request")
			},
		},
		"EmptyAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "valid token not found in request")
			},
		},
		"EmptySpaceAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", " ")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "valid token not found in request")
			},
		},
		"EmptyCookieAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				r.AddCookie(&http.Cookie{
					Name:     CookieAuthorizationName,
					Value:    "",
					MaxAge:   int(time.Until(time.Now().Add(time.Minute * 1)).Seconds()),
					Path:     CookiePath,
					Domain:   CookieDomain,
					SameSite: http.SameSiteStrictMode,
					Secure:   CookieSecureHTTPSOnly,
					HttpOnly: CookieHTTPOnlyNotJavascriptAccessible,
				})

				r.Header.Add("Authorization", " ")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "skipped validating empty token")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			db := setupDB(t)

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("db", db)

			authFunc, ok := v["authFunc"].(func(*testing.T, *gorm.DB, *gin.Context))
			require.True(t, ok)
			authFunc(t, db, c)

			err := RequireAccessKey(c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *gin.Context, error))
			require.True(t, ok)

			verifyFunc(t, c, err)
		})
	}
}
