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
		"ValidEmailAndOneTimePasswordFirstUse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "bruce@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				assert.NilError(t, err)

				return email, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "bruce@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, requiresUpdate)
			},
		},
		"ValidEmailAndOneTimePasswordFailsOnReuse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "barbra@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				assert.NilError(t, err)

				_, _, _, err = LoginWithUserCredential(c, email, oneTimePassword, time.Now().Add(time.Hour))
				assert.NilError(t, err)

				return email, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.ErrorContains(t, err, "one time password cannot be used more than once")
			},
		},
		"ValidEmailAndValidSpecifiedPassword": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "selina@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
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

				return email, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "selina@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, !requiresUpdate)
			},
		},
		"ValidEmailAndValidSpecifiedPasswordCanBeReused": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "penguin@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
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

				_, _, _, err = LoginWithUserCredential(c, email, "password", time.Now().Add(time.Hour))
				assert.NilError(t, err)

				return email, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "penguin@example.com", user.Name)
				assert.Assert(t, len(secret) != 0)
				assert.Assert(t, !requiresUpdate)
			},
		},
		"ValidEmailAndInvalidPasswordFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "gordon@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
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

				return email, "wrong_password"
			},
			"verify": func(t *testing.T, secret string, user *models.Identity, requiresUpdate bool, err error) {
				assert.ErrorContains(t, err, "password verify")
			},
		},
		"ValidEmailAndNoCredentialsFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "edward@example.com"
				user := &models.Identity{Name: email, Kind: models.UserKind}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				return email, ""
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
			email, password := setupFunc(t, c, db)

			secret, user, requiresUpdate, err := LoginWithUserCredential(c, email, password, time.Now().Add(time.Hour))

			verifyFunc, ok := v["verify"].(func(*testing.T, string, *models.Identity, bool, error))
			assert.Assert(t, ok)

			verifyFunc(t, secret, user, requiresUpdate, err)
		})
	}
}
