package authn

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestKeyExchangeAuthentication(t *testing.T) {
	db := setupDB(t)

	type testCase struct {
		setup       func(t *testing.T, db *gorm.DB) LoginMethod
		expectedErr string
		expected    func(t *testing.T, identity *models.Identity, provider *models.Provider)
	}

	cases := map[string]testCase{
		"InvalidAccessKeyCannotBeExchanged": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "goku@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				invalidKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"

				return NewKeyExchangeAuthentication(invalidKey, time.Now().Add(5*time.Minute))
			},
			expectedErr: "could not get access key from database",
		},
		"ExpiredAccessKeyCannotBeExchanged": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "bulma@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:       "expired-key",
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(db).ID,
					ExpiresAt:  time.Now().Add(-1 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, time.Now().Add(5*time.Minute))
			},
			expectedErr: data.ErrAccessKeyExpired.Error(),
		},
		"AccessKeyCannotBeExchangedForLongerLived": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "krillin@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:       "krillins-key",
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(db).ID,
					ExpiresAt:  time.Now().Add(1 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, time.Now().Add(5*time.Minute))
			},
			expectedErr: "cannot exchange an access key for another access key with a longer lifetime",
		},
		"AccessKeyCannotBeExchangedWhenUserNoLongerExists": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				key := &models.AccessKey{
					Name:       "no-user-key",
					IssuedFor:  uid.New(), // simulate the user not existing
					ProviderID: data.InfraProvider(db).ID,
					ExpiresAt:  time.Now().Add(5 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, time.Now().Add(1*time.Minute))
			},
			expectedErr: "user is not valid",
		},
		"ValidAccessKeySuccess": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "cell@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:       "cell-key",
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(db).ID,
					ExpiresAt:  time.Now().Add(5 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, time.Now().Add(1*time.Minute))
			},
			expected: func(t *testing.T, identity *models.Identity, provider *models.Provider) {
				assert.Equal(t, identity.Name, "cell@example.com")
				assert.Equal(t, data.InfraProvider(db).ID, provider.ID)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			keyExchangeLogin := tc.setup(t, db)

			identity, provider, _, err := keyExchangeLogin.Authenticate(context.Background(), db)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}

			assert.NilError(t, err)
			tc.expected(t, identity, provider)
		})
	}
}
