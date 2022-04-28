package access

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestLoginWithUserCredential(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	cases := map[string]map[string]interface{}{
		"ValidUsernameAndOneTimePasswordFirstUse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "bruce@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				assert.NilError(t, err)

				return username, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "bruce@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, requiresUpdate)
			},
		},
		"ValidUsernameAndOneTimePasswordFailsOnReuse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "barbra@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				assert.NilError(t, err)

				_, _, _, err = LoginWithPasswordCredential(c, username, oneTimePassword, time.Now().Add(time.Hour))
				assert.NilError(t, err)

				return username, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.ErrorContains(t, err, "one time password cannot be used more than once")
			},
		},
		"ValidUsernameAndValidSpecifiedPassword": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "selina@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				assert.NilError(t, err)

				userCredential := &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				assert.NilError(t, err)

				return username, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "selina@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, !requiresUpdate)
			},
		},
		"ValidUsernameAndValidSpecifiedPasswordCanBeReused": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "penguin@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				assert.NilError(t, err)

				userCredential := &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				assert.NilError(t, err)

				_, _, _, err = LoginWithPasswordCredential(c, username, "password", time.Now().Add(time.Hour))
				assert.NilError(t, err)

				return username, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "penguin@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, !requiresUpdate)
			},
		},
		"ValidUsernameAndInvalidPasswordFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "gordon@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				assert.NilError(t, err)

				userCredential := &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				assert.NilError(t, err)

				return username, "wrong_password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.ErrorContains(t, err, "password verify")
			},
		},
		"ValidUsernameAndNoCredentialsFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				username := "edward@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				return username, ""
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gin.Context, *gorm.DB) (string, string))
			assert.Assert(t, ok)
			username, password := setupFunc(t, c, db)

			secret, user, requiresUpdate, err := LoginWithPasswordCredential(c, username, password, time.Now().Add(time.Hour))

			verifyFunc, ok := v["verify"].(func(*testing.T, string, *models.Identity, bool, error))
			assert.Assert(t, ok)

			verifyFunc(t, secret, user, requiresUpdate, err)
		})
	}
}
