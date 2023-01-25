package authn

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestKeyExchangeAuthentication(t *testing.T) {
	tx := setupDB(t)

	type testCase struct {
		setup       func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time)
		expectedErr string
		expected    func(t *testing.T, authnIdentity AuthenticatedIdentity)
	}

	shortExpiry := time.Now().UTC().Add(1 * time.Minute)
	longExpiry := time.Now().UTC().Add(30 * 24 * time.Hour)
	threshold := opt.TimeWithThreshold(time.Second)

	cases := map[string]testCase{
		"InvalidAccessKeyCannotBeExchanged": {
			setup: func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time) {
				user := &models.Identity{Name: "goku@example.com"}
				err := data.CreateIdentity(tx, user)
				assert.NilError(t, err)

				invalidKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"

				return NewKeyExchangeAuthentication(invalidKey), time.Now().Add(5 * time.Minute)
			},
			expectedErr: "could not get access key from database",
		},
		"ExpiredAccessKeyCannotBeExchanged": {
			setup: func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time) {
				user := &models.Identity{Name: "bulma@example.com"}
				err := data.CreateIdentity(tx, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:        "expired-key",
					IssuedForID: user.ID,
					ProviderID:  data.InfraProvider(tx).ID,
					ExpiresAt:   time.Now().Add(-1 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(tx, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer), time.Now().Add(5 * time.Minute)
			},
			expectedErr: data.ErrAccessKeyExpired.Error(),
		},
		"AccessKeyCannotBeExchangedWhenUserNoLongerExists": {
			setup: func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time) {
				user := &models.Identity{Name: "notforlong@example.com"}
				err := data.CreateIdentity(tx, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:        "no-user-key",
					IssuedForID: user.ID,
					ProviderID:  data.InfraProvider(tx).ID,
					ExpiresAt:   time.Now().Add(5 * time.Minute),
				}

				bearer, err := data.CreateAccessKey(tx, key)
				assert.NilError(t, err)

				assert.NilError(t, data.DeleteIdentities(tx, data.DeleteIdentitiesOptions{ByID: user.ID, ByProviderID: data.InfraProvider(tx).ID}))

				return NewKeyExchangeAuthentication(bearer), time.Now().Add(5 * time.Minute)
			},
			expectedErr: "record not found",
		},
		"AccessKeyCannotBeExchangedForLongerLived": {
			setup: func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time) {
				user := &models.Identity{Name: "krillin@example.com"}
				err := data.CreateIdentity(tx, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:        "krillins-key",
					IssuedForID: user.ID,
					ProviderID:  data.InfraProvider(tx).ID,
					ExpiresAt:   shortExpiry,
				}

				bearer, err := data.CreateAccessKey(tx, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer), longExpiry
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, authnIdentity.Identity.Name, "krillin@example.com")
				assert.Equal(t, data.InfraProvider(tx).ID, authnIdentity.Provider.ID)
				assert.DeepEqual(t, authnIdentity.SessionExpiry, shortExpiry, threshold)
			},
		},
		"ValidAccessKeySuccess": {
			setup: func(t *testing.T, tx data.WriteTxn) (LoginMethod, time.Time) {
				user := &models.Identity{Name: "cell@example.com"}
				err := data.CreateIdentity(tx, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					Name:        "cell-key",
					IssuedForID: user.ID,
					ProviderID:  data.InfraProvider(tx).ID,
					ExpiresAt:   time.Now().Add(31 * 24 * time.Hour),
				}

				bearer, err := data.CreateAccessKey(tx, key)
				assert.NilError(t, err)

				return NewKeyExchangeAuthentication(bearer), longExpiry
			},
			expected: func(t *testing.T, authnIdentity AuthenticatedIdentity) {
				assert.Equal(t, authnIdentity.Identity.Name, "cell@example.com")
				assert.Equal(t, data.InfraProvider(tx).ID, authnIdentity.Provider.ID)
				// if the request expiry is less than the lifetime of the requesting key,
				// the issued key should match the requested expiry
				assert.DeepEqual(t, authnIdentity.SessionExpiry, longExpiry, threshold)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			keyExchangeLogin, exp := tc.setup(t, tx)

			authnIdentity, err := keyExchangeLogin.Authenticate(context.Background(), tx, exp)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}

			assert.NilError(t, err)
			tc.expected(t, authnIdentity)
		})
	}
}
