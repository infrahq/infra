package authn

import (
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

	cases := map[string]map[string]interface{}{
		"InvalidAccessKeyCannotBeExchanged": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
				user := &models.Identity{Name: "goku@example.com"}
				err := data.CreateIdentity(db, user)
				assert.NilError(t, err)

				invalidKey := "aaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbb"

				return NewKeyExchangeAuthentication(invalidKey, time.Now().Add(5*time.Minute))
			},
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "could not get access key from database")
			},
		},
		"ExpiredAccessKeyCannotBeExchanged": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
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
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "token expired")
			},
		},
		"AccessKeyCannotBeExchangedForLongerLived": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
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
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "cannot exchange an access key for another access key with a longer lifetime")
			},
		},
		"AccessKeyCannotBeExchangedWhenUserNoLongerExists": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
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
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.ErrorContains(t, err, "user is not valid")
			},
		},
		"ValidAccessKeySuccess": {
			"setup": func(t *testing.T, db *gorm.DB) LoginMethod {
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
			"verify": func(t *testing.T, identity *models.Identity, provider *models.Provider, err error) {
				assert.NilError(t, err)
				assert.Equal(t, identity.Name, "cell@example.com")
				assert.Equal(t, data.InfraProvider(db).ID, provider.ID)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			setupFunc, ok := v["setup"].(func(*testing.T, *gorm.DB) LoginMethod)
			assert.Assert(t, ok)
			keyExchangeLogin := setupFunc(t, db)

			identity, provider, err := keyExchangeLogin.Authenticate(db)

			verifyFunc, ok := v["verify"].(func(*testing.T, *models.Identity, *models.Provider, error))
			assert.Assert(t, ok)

			verifyFunc(t, identity, provider, err)
		})
	}
}
