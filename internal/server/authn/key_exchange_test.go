package authn

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestKeyExchangeAuthentication(t *testing.T) {
	db := setupDB(t)

	type testCase struct {
		setup       func(t *testing.T, db *gorm.DB) LoginMethod
		expectedErr string
		expected    func(t *testing.T, authnIdentity AuthenticatedIdentity)
	}

	shortExpiry := time.Now().Add(1 * time.Minute)
	longExpiry := time.Now().Add(30 * 24 * time.Hour)

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
		"AccessKeyCannotBeExchangedWhenUserNoLongerExists": {
			setup: func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "notforlong@example.com"}
				user.DeletedAt.Time = time.Now()
				user.DeletedAt.Valid = true
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:       "no-user-key",
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(db).ID,
					ExpiresAt:  time.Now().Add(5 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, time.Now().Add(1*time.Minute))
			},
			expectedErr: "user is not valid",
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
					ExpiresAt:  shortExpiry,
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, longExpiry)
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, authnIdentity.Identity.Name, "krillin@example.com")
				assert.Equal(t, data.InfraProvider(db).ID, authnIdentity.Provider.ID)
				assert.Assert(t, authnIdentity.SessionExpiry.Equal(shortExpiry))
			},
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
					ExpiresAt:  time.Now().Add(31 * 24 * time.Hour),
				}

				bearer, err := data.CreateAccessKey(db, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer, longExpiry)
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, authnIdentity.Identity.Name, "cell@example.com")
				assert.Equal(t, data.InfraProvider(db).ID, authnIdentity.Provider.ID)
				// if the request expiry is less than the lifetime of the requesting key,
				// the issued key should match the requested expiry
				assert.Assert(t, authnIdentity.SessionExpiry.Equal(longExpiry))
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			keyExchangeLogin := tc.setup(t, db)

			authnIdentity, err := keyExchangeLogin.Authenticate(context.Background(), db, longExpiry)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}

			assert.NilError(t, err)
			tc.expected(t, authnIdentity)
		})
	}
}
