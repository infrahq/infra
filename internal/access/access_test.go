package access

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	require.NoError(t, err)

	db, err := data.NewDB(driver)
	require.NoError(t, err)

	return db
}

func issueToken(t *testing.T, db *gorm.DB, email, permissions string, sessionDuration time.Duration) string {
	user, err := data.CreateUser(db, &models.User{Email: email, Permissions: permissions})
	require.NoError(t, err)

	token := &models.Token{
		User:            *user,
		SessionDuration: sessionDuration,
	}
	token, err = data.CreateToken(db, token)
	require.NoError(t, err)

	return token.SessionToken()
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
				authentication := issueToken(t, db, "existing@infrahq.com", string(PermissionAPITokenCreate), time.Minute*1)
				// user permissions updated after token is issued
				_, err := data.CreateOrUpdateUser(db, &models.User{Email: "existing@infrahq.com", Permissions: string(PermissionCredentialCreate)}, &models.User{Email: "existing@infrahq.com"})
				require.NoError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(PermissionCredentialCreate), permissions)
			},
		},
		"TokenMultiplePermissionsUpdatedToMatchParentUser": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionAPITokenCreate), string(PermissionCredentialCreate)}
				authentication := issueToken(t, db, "existing@infrahq.com", strings.Join(permissions, " "), time.Minute*1)
				// user permissions updated after token is issued
				_, err := data.CreateOrUpdateUser(db, &models.User{Email: "existing@infrahq.com", Permissions: string(PermissionCredentialCreate)}, &models.User{Email: "existing@infrahq.com"})
				require.NoError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				require.NoError(t, err)
				permissions, ok := c.Get("permissions")
				require.True(t, ok)
				require.Equal(t, string(PermissionCredentialCreate), permissions)
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

func TestRequireAuthorization(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"AuthorizedAll": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAll))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"AuthorizedAllAlternate": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllAlternate))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"AuthorizedExactMatch": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUserRead))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"AuthorizedOneOfMany": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserRead), string(PermissionUserCreate), string(PermissionUserDelete)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"AuthorizedWildcardAction": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionUser))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"AuthorizedWildcardResource": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("permissions", string(PermissionAllRead))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APITokenAuthorizedNotFirst": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionProviderRead), string(PermissionUserRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APITokenAuthorizedNotLast": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionGroupRead), string(PermissionUserRead), string(PermissionProviderRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APITokenAuthorizedNoMatch": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				permissions := []string{string(PermissionUserCreate), string(PermissionGroupRead)}
				c.Set("permissions", strings.Join(permissions, " "))
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "forbidden")
			},
		},
		"NotRequired": {
			"permission": Permission(""),
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
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

			permission, ok := v["permission"].(Permission)
			require.True(t, ok)
			_, err := RequireAuthorization(c, permission)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, error))
			require.True(t, ok)

			verifyFunc(t, err)
		})
	}
}

func TestAllRequired(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"ExactMatch": {
			"permissions": []string{string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"SubsetMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserRead)},
			"expected":    true,
		},
		"NoMatch": {
			"permissions": []string{string(PermissionUserCreate), string(PermissionUserRead)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"NoPermissions": {
			"permissions": []string{},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    false,
		},
		"AllPermissions": {
			"permissions": []string{string(PermissionAll)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsAlternate": {
			"permissions": []string{string(PermissionAllAlternate)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
		"AllPermissionsForResource": {
			"permissions": []string{string(PermissionUser)},
			"required":    []string{string(PermissionUserDelete)},
			"expected":    true,
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			permissions, ok := v["permissions"].([]string)
			require.True(t, ok)

			required, ok := v["required"].([]string)
			require.True(t, ok)

			result := AllRequired(permissions, required)

			expected, ok := v["expected"].(bool)
			require.True(t, ok)

			assert.Equal(t, expected, result)
		})
	}
}
