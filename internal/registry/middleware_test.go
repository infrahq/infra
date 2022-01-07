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
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRequestTimeoutError(t *testing.T) {
	requestTimeout = 100 * time.Millisecond

	router := gin.New()
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
	router.Use(RequestTimeoutMiddleware())
	router.GET("/", func(c *gin.Context) {
		require.NoError(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequireAuthentication(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"TokenValid": {
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
		"TokenSinglePermissionUpdatedToMatchParentUser": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueToken(t, db, "existing@infrahq.com", string(access.PermissionAPITokenCreate), time.Minute*1)
				// user permissions updated after token is issued
				_, err := data.CreateOrUpdateUser(db, &models.User{Email: "existing@infrahq.com", Permissions: string(access.PermissionCredentialCreate)}, &models.User{Email: "existing@infrahq.com"})
				require.NoError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(access.PermissionCredentialCreate), permissions)
			},
		},
		"TokenMultiplePermissionsUpdatedToMatchParentUser": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(access.PermissionAPITokenCreate), string(access.PermissionCredentialCreate)}
				authentication := issueToken(t, db, "existing@infrahq.com", strings.Join(permissions, " "), time.Minute*1)
				// user permissions updated after token is issued
				_, err := data.CreateOrUpdateUser(db, &models.User{Email: "existing@infrahq.com", Permissions: string(access.PermissionCredentialCreate)}, &models.User{Email: "existing@infrahq.com"})
				require.NoError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(access.PermissionCredentialCreate), permissions)
			},
		},
		"TokenExpired": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*-1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "rejected token: token expired")
			},
		},
		"TokenInvalidKey": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				secret := token[models.TokenKeyLength:]
				authentication := fmt.Sprintf("%s%s", generate.MathRandom(models.TokenKeyLength), secret)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "could not get token from database, it may not exist: record not found")
			},
		},
		"TokenNoMatch": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := generate.MathRandom(models.TokenLength)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "could not get token from database, it may not exist: record not found")
			},
		},
		"TokenInvalidSecret": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				key := token[:models.TokenKeyLength]
				secret, err := generate.CryptoRandom(models.TokenSecretLength)
				require.NoError(t, err)
				authentication := fmt.Sprintf("%s%s", key, secret)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "rejected invalid token: token invalid secret")
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
				require.EqualError(t, err, "rejected token of invalid length")
			},
		},
		"NoAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				// nil pointer if we don't seup the request header here
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "valid token not found in authorization header, expecting the format `Bearer $token`")
			},
		},
		"EmptyAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.EqualError(t, err, "valid token not found in authorization header, expecting the format `Bearer $token`")
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

			err := RequireAuthentication(c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *gin.Context, error))
			require.True(t, ok)

			verifyFunc(t, c, err)
		})
	}
}

func issueToken(t *testing.T, db *gorm.DB, email, permissions string, sessionDuration time.Duration) string {
	user, err := data.CreateUser(db, &models.User{Email: email, Permissions: permissions})
	require.NoError(t, err)

	token := &models.Token{
		UserID:          user.ID,
		SessionDuration: sessionDuration,
	}
	token, err = data.CreateToken(db, token)
	require.NoError(t, err)

	return token.SessionToken()
}
