package access

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/generate"
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

func issueAPIKey(t *testing.T, db *gorm.DB, permissions string) string {
	secret, err := generate.CryptoRandom(data.APIKeyLength)
	require.NoError(t, err)

	apiKey := &data.APIKey{
		Name:        "test",
		Key:         secret,
		Permissions: permissions,
	}

	apiKey, err = data.CreateAPIKey(db, apiKey)
	require.NoError(t, err)

	return apiKey.Key
}

func issueToken(t *testing.T, db *gorm.DB, email, permissions string, sessionDuration time.Duration) string {
	user, err := data.CreateUser(db, &data.User{Email: email})
	require.NoError(t, err)

	token := &data.Token{
		User:            *user,
		SessionDuration: sessionDuration,
		Permissions:     permissions,
	}
	token, err = data.CreateToken(db, token)
	require.NoError(t, err)

	return token.SessionToken()
}

func TestRequireAuthorization(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"TokenAuthorizedAll": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"TokenExpired": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*-1)
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "token expired")
			},
		},
		"TokenInvalidKey": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				secret := token[data.TokenKeyLength:]
				authorization := fmt.Sprintf("%s%s", generate.MathRandom(data.TokenKeyLength), secret)
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "token invalid")
			},
		},
		"TokenNoMatch": {
			"permission": PermissionAPIKeyList,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueToken(t, db, "existing@infrahq.com", "infra.user.read", time.Minute*1)
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "forbidden")
			},
		},
		"TokenInvalidSecret": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", "*", time.Minute*1)
				key := token[:data.TokenKeyLength]
				secret, err := generate.CryptoRandom(data.TokenSecretLength)
				require.NoError(t, err)
				authorization := fmt.Sprintf("%s%s", key, secret)
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "token invalid")
			},
		},
		"APIKeyAuthorizedAll": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "*")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedAllAlternate": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.*")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedExactMatch": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.user.read")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedOneOfMany": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.user.read infra.user.create infra.user.delete")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedWildcardAction": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.user.*")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedWildcardResource": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.*.read")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedNotFirst": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.group.read infra.provider.read infra.user.read")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyAuthorizedNotLast": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.group.read infra.user.read infra.provider.read")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		"APIKeyNoMatch": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization := issueAPIKey(t, db, "infra.user.create infra.group.read")
				c.Set("authorization", authorization)
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "forbidden")
			},
		},
		"UnknownAuthorizationMethod": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authorization, err := generate.CryptoRandom(32)
				require.NoError(t, err)

				c.Set("authorization", authorization)
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
		"NoAuthorization": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "token invalid")
			},
		},
		"EmptyAuthorization": {
			"permission": PermissionUserRead,
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				c.Set("authorization", "")
			},
			"verifyFunc": func(t *testing.T, err error) {
				require.EqualError(t, err, "token invalid")
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
			_, _, err := RequireAuthorization(c, permission)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, error))
			require.True(t, ok)

			verifyFunc(t, err)
		})
	}
}
