package registry

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	return db
}

func issueToken(t *testing.T, db *gorm.DB, email, permissions string, sessionDuration time.Duration) string {
	user := &models.User{Email: email, Permissions: permissions}

	err := data.CreateUser(db, user)
	require.NoError(t, err)

	token := &models.AccessKey{
		UserID:      user.ID,
		Permissions: permissions,
		ExpiresAt:   time.Now().Add(sessionDuration),
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
				authentication := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
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
		"AccessKeySetsPermissions": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueToken(t, db, "existing@infrahq.com", string(access.PermissionTokenCreate), time.Minute*1)
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
				authentication := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*-1)
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
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
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
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
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
				require.Contains(t, err.Error(), "valid token not found in authorization header, expecting the format `Bearer $token`")
			},
		},
		"EmptyAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.Contains(t, err.Error(), "valid token not found in authorization header, expecting the format `Bearer $token`")
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
