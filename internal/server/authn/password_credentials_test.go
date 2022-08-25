package authn

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestPasswordCredentialAuthentication(t *testing.T) {
	db := setupDB(t)

	type testCase struct {
		setup       func(t *testing.T, db data.GormTxn) LoginMethod
		expectedErr string
		expected    func(t *testing.T, authnIdentity AuthenticatedIdentity)
	}

	cases := map[string]testCase{
		"UsernameAndOneTimePasswordFirstUse": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "goku@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				oneTimePassword := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(oneTimePassword), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    hash,
					OneTimePassword: true,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, oneTimePassword)
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, "goku@example.com", authnIdentity.Identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, authnIdentity.Provider.Name)
				assert.Equal(t, models.ProviderKindInfra, authnIdentity.Provider.Kind)
			},
		},
		"UsernameAndPassword": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "bulma@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    hash,
					OneTimePassword: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, password)
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, "bulma@example.com", authnIdentity.Identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, authnIdentity.Provider.Name)
				assert.Equal(t, models.ProviderKindInfra, authnIdentity.Provider.Kind)
			},
		},
		"UsernameAndPasswordReuse": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "cell@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    hash,
					OneTimePassword: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				userPassLogin := NewPasswordCredentialAuthentication(username, password)

				_, err = userPassLogin.Authenticate(context.Background(), db, time.Now().Add(1*time.Minute))
				assert.NilError(t, err)

				return userPassLogin
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, "cell@example.com", authnIdentity.Identity.Name)
				assert.Equal(t, models.InternalInfraProviderName, authnIdentity.Provider.Name)
				assert.Equal(t, models.ProviderKindInfra, authnIdentity.Provider.Kind)
			},
		},
		"ValidUsernameAndNoPasswordFails": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "krillin@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "")
			},
			expectedErr: "record not found",
		},
		"UsernameAndInvalidPasswordFails": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "po@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    hash,
					OneTimePassword: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "invalidPassword")
			},
			expectedErr: "hashedPassword is not the hash of the given password",
		},
		"UsernameAndEmptyPasswordFails": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				username := "gohan@example.com"
				user := &models.Identity{Name: username}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				password := "password123"
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				assert.NilError(t, err)

				creds := models.Credential{
					IdentityID:      user.ID,
					PasswordHash:    hash,
					OneTimePassword: false,
				}

				err = data.CreateCredential(db, &creds)
				assert.NilError(t, err)

				return NewPasswordCredentialAuthentication(username, "")
			},
			expectedErr: "hashedPassword is not the hash of the given password",
		},
		"EmptyUsernameAndPasswordFails": {
			setup: func(t *testing.T, db data.GormTxn) LoginMethod {
				return NewPasswordCredentialAuthentication("", "whatever")
			},
			expectedErr: "record not found",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			credentialLogin := tc.setup(t, db)

			authnIdentity, err := credentialLogin.Authenticate(context.Background(), db, time.Now().Add(1*time.Minute))
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			assert.NilError(t, err)
			tc.expected(t, authnIdentity)
		})
	}
}
