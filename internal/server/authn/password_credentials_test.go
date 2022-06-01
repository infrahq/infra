package authn

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestPasswordCredentialAuthentication(t *testing.T) {
	db := setupDB(t)

	cases := map[string]map[string]interface{}{
		"UsernameAndOneTimePasswordFirstUse": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "goku@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     true,
					OneTimePasswordUsed: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, oneTimePassword)
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "goku@example.com", identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, provider.Name)
			},
		},
		"UsernameAndOneTimePasswordFailsOnReuse": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "vegeta@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     true,
					OneTimePasswordUsed: true,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, oneTimePassword)
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "one time password cannot be used more than once")
			},
		},
		"UsernameAndPassword": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "bulma@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     false,
					OneTimePasswordUsed: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, password)
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "bulma@example.com", identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, provider.Name)
			},
		},
		"UsernameAndPasswordReuse": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "cell@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     false,
					OneTimePasswordUsed: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				userPassLogin := NewPasswordCredentialAuthentication(username, password)

				_, _, err = userPassLogin.Authenticate(db)
				assert.NilError(t, err)

				return userPassLogin
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.NilError(t, err)
				assert.Equal(t, "cell@example.com", identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, provider.Name)
			},
		},
		"ValidUsernameAndNoPasswordFails": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "krillin@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "")
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
		"UsernameAndInvalidPasswordFails": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "po@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     false,
					OneTimePasswordUsed: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "invalidPassword")
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "hashedPassword is not the hash of the given password")
			},
		},
		"UsernameAndEmptyPasswordFails": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				username := "gohan@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:          user.ID,
					PasswordHash:        hash,
					OneTimePassword:     false,
					OneTimePasswordUsed: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "")
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "hashedPassword is not the hash of the given password")
			},
		},
		"EmptyUsernameAndPasswordFails": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				return NewPasswordCredentialAuthentication("", "whatever")
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "record not found")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) LoginMethod)
			assert.Assert(t, ok)
			credentialLogin := setupFunc(t, db)

			identity, provider, err := credentialLogin.Authenticate(db)

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.Identity, *models.Provider, error))
			assert.Assert(t, ok)

			verifyFunc(t, identity, provider, err)
		})
	}
}
