package access

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestLoginWithUserCredential(t *testing.T) {
	// setup db and context
	db := setupDB(t)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)

	admin := &models.User{Email: "admin@example.com"}
	err := data.CreateUser(db, admin)
	require.NoError(t, err)

	c.Set("identity", admin.PolyID())

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(db, adminGrant)
	require.NoError(t, err)

	SetupTestSecretProvider(t)

	provider := &models.Provider{Name: models.InternalInfraProviderName}
	err = data.CreateProvider(db, provider)
	require.NoError(t, err)

	cases := map[string]map[string]interface{}{
		"ValidEmailAndOneTimePasswordFirstUse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "bruce@example.com"
				user := &models.User{Email: email, ProviderID: provider.ID}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				require.NoError(t, err)

				return email, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, "bruce@example.com", user.Email)
				require.NotEmpty(t, secret)
				require.True(t, requiresUpdate)
			},
		},
		"ValidEmailAndOneTimePasswordFailsOnReuse": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "barbra@example.com"
				user := &models.User{Email: email, ProviderID: provider.ID}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				oneTimePassword, err := CreateCredential(c, *user)
				require.NoError(t, err)

				_, _, _, err = LoginWithUserCredential(c, email, oneTimePassword, time.Now().Add(time.Hour))
				require.NoError(t, err)

				return email, oneTimePassword
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.Error(t, err, "one time password cannot be used more than once")
			},
		},
		"ValidEmailAndValidSpecifiedPassword": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "selina@example.com"
				user := &models.User{Email: email, ProviderID: provider.ID}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
				require.NoError(t, err)

				userCredential := &models.Credential{
					Identity:     user.PolyID(),
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				require.NoError(t, err)

				return email, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, "selina@example.com", user.Email)
				require.NotEmpty(t, secret)
				require.False(t, requiresUpdate)
			},
		},
		"ValidEmailAndValidSpecifiedPasswordCanBeReused": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "penguin@example.com"
				user := &models.User{Email: email, ProviderID: provider.ID}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
				require.NoError(t, err)

				userCredential := &models.Credential{
					Identity:     user.PolyID(),
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				require.NoError(t, err)

				_, _, _, err = LoginWithUserCredential(c, email, "password", time.Now().Add(time.Hour))
				require.NoError(t, err)

				return email, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.NoError(t, err)
				require.Equal(t, "penguin@example.com", user.Email)
				require.NotEmpty(t, secret)
				require.False(t, requiresUpdate)
			},
		},
		"ValidEmailAndInvalidPasswordFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "gordon@example.com"
				user := &models.User{Email: email, ProviderID: provider.ID}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
				require.NoError(t, err)

				userCredential := &models.Credential{
					Identity:     user.PolyID(),
					PasswordHash: hash,
				}

				err = data.CreateCredential(db, userCredential)
				require.NoError(t, err)

				return email, "wrong_password"
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.Error(t, err, "password verify")
			},
		},
		"ValidEmailAndNotInfraProviderFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "edward@example.com"
				user := &models.User{Email: email, ProviderID: uid.New()}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
				require.NoError(t, err)

				userCredential := &models.Credential{
					Identity:     user.PolyID(),
					PasswordHash: hash,
				}

				// this shouldnt be possible for non-infra providers
				err = data.CreateCredential(db, userCredential)
				require.NoError(t, err)

				return email, "password"
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.Error(t, err, "record not found")
			},
		},
		"ValidEmailAndNoCredentialsFails": {
			"setup": func(t *testing.T, c *gin.Context, db *gorm.DB) (string, string) {
				email := "edward@example.com"
				user := &models.User{Email: email, ProviderID: uid.New()}
				err := data.CreateUser(db, user)
				require.NoError(t, err)

				return email, ""
			},
			"verify": func(t *testing.T, secret string, user *models.User, requiresUpdate bool, err error) {
				require.Error(t, err, "record not found")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gin.Context, *gorm.DB) (string, string))
			require.True(t, ok)
			email, password := setupFunc(t, c, db)

			secret, user, requiresUpdate, err := LoginWithUserCredential(c, email, password, time.Now().Add(time.Hour))

			verifyFunc, ok := v["verify"].(func(*testing.T, string, *models.User, bool, error))
			require.True(t, ok)

			verifyFunc(t, secret, user, requiresUpdate, err)
		})
	}
}
