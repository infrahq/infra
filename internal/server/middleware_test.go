package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	tpatch "github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	tpatch.ModelsSymmetricKey(t)
	db, err := data.NewDB(driver, nil)
	assert.NilError(t, err)

	// create the provider if it's missing.
	data.InfraProvider(db)

	err = data.SaveSettings(db, &models.Settings{})
	assert.NilError(t, err)

	return db
}

func issueToken(t *testing.T, db *gorm.DB, identityName string, sessionDuration time.Duration) string {
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
	db := setupDB(t)
	var ctx context.Context
	var cancel context.CancelFunc

	router := gin.New()
	router.Use(
		func(c *gin.Context) {
			// this is a custom copy of the timeout middleware so I can grab and control the cancel() func. Otherwise the test is too flakey with timing race conditions.
			ctx, cancel = context.WithTimeout(c, 100*time.Millisecond)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)
			c.Set("ctx", ctx)
			c.Next()
		},
		DatabaseMiddleware(db),
	)
	router.GET("/", func(c *gin.Context) {
		db, ok := c.MustGet("db").(*gorm.DB)
		assert.Check(t, ok)
		cancel()
		err := db.Exec("select 1;").Error
		assert.Error(t, err, "context canceled")

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequireAuthentication(t *testing.T) {
	cases := map[string]map[string]interface{}{
		"AccessKeyValid": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.NilError(t, err)
			},
		},
		"AccessKeyExpired": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := issueToken(t, db, "existing@infrahq.com", time.Minute*-1)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorIs(t, err, data.ErrAccessKeyExpired)
			},
		},
		"AccessKeyInvalidKey": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				secret := token[:models.AccessKeySecretLength]
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), secret)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		"AccessKeyNoMatch": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication := fmt.Sprintf("%s.%s", uid.New().String(), generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		"AccessKeyInvalidSecret": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				token := issueToken(t, db, "existing@infrahq.com", time.Minute*1)
				authentication := fmt.Sprintf("%s.%s", strings.Split(token, ".")[0], generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric))
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "access key invalid secret")
			},
		},
		"UnknownAuthenticationMethod": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				authentication, err := generate.CryptoRandom(32, generate.CharsetAlphaNumeric)
				assert.NilError(t, err)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "Bearer "+authentication)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "invalid access key format")
			},
		},
		"NoAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				// nil pointer if we don't seup the request header here
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
			},
		},
		"EmptyAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", "")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
			},
		},
		"EmptySpaceAuthentication": {
			"authFunc": func(t *testing.T, db *gorm.DB, c *gin.Context) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Add("Authorization", " ")
				c.Request = r
			},
			"verifyFunc": func(t *testing.T, c *gin.Context, err error) {
				assert.ErrorContains(t, err, "valid token not found in request")
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
				assert.ErrorContains(t, err, "skipped validating empty token")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			db := setupDB(t)

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("db", db)

			authFunc, ok := v["authFunc"].(func(*testing.T, *gorm.DB, *gin.Context))
			assert.Assert(t, ok)
			authFunc(t, db, c)

			err := RequireAccessKey(c)

			verifyFunc, ok := v["verifyFunc"].(func(*testing.T, *gin.Context, error))
			assert.Assert(t, ok)

			verifyFunc(t, c, err)
		})
	}
}

func TestDestinationMiddleware(t *testing.T) {
	db := setupDB(t)

	router := gin.New()
	router.Use(
		DatabaseMiddleware(db),
		AuthenticationMiddleware(),
		DestinationMiddleware(),
	)

	connector := models.Identity{Name: "connector"}
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

	router.GET("/good", func(c *gin.Context) {
		assert.Equal(t, c.Request.Method, http.MethodGet)
		assert.Equal(t, c.Request.URL.Path, "/good")
	})

	t.Run("good", func(t *testing.T) {
		destination := &models.Destination{Name: t.Name(), UniqueID: t.Name()}
		err := data.CreateDestination(db, destination)
		assert.NilError(t, err)

		r := httptest.NewRequest("GET", "/good", nil)
		r.Header.Add("Infra-Destination", destination.UniqueID)
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", secret))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

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
		router.ServeHTTP(w, r)

		destination, err = data.GetDestination(db, data.ByOptionalUniqueID(destination.UniqueID))
		assert.NilError(t, err)
		assert.Equal(t, destination.LastSeenAt, time.Time{})
	})

	t.Run("good no destination", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/good", nil)
		r.Header.Add("Infra-Destination", "nonexistent")
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", secret))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		_, err := data.GetDestination(db, data.ByOptionalUniqueID("nonexistent"))
		assert.ErrorIs(t, err, internal.ErrNotFound)
	})
}
