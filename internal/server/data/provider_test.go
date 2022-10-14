package data

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		providerDevelop := models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}

		err := db.Create(&providerDevelop).Error
		assert.NilError(t, err)

		var provider models.Provider
		err = db.Not("name = ?", models.InternalInfraProviderName).First(&provider).Error
		assert.NilError(t, err)
		assert.Equal(t, "example.com", provider.URL)
	})
}

func createProviders(t *testing.T, db GormTxn, providers ...*models.Provider) {
	for i := range providers {
		err := CreateProvider(db, providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProvider_DuplicateName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		providerDevelop.ID = 0 // zero out the ID so that the conflict is on name
		err := CreateProvider(db, &providerDevelop)

		var uniqueConstraintErr UniqueConstraintError
		assert.Assert(t, errors.As(err, &uniqueConstraintErr), "error is wrong type %T", err)
		expected := UniqueConstraintError{Column: "name", Table: "providers"}
		assert.DeepEqual(t, uniqueConstraintErr, expected)
	})
}

func TestCreateProvider_RecreateWithDuplicateDomain(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		err := DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
		assert.NilError(t, err)

		err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta})
		assert.NilError(t, err)
	})
}

// TODO: combine CreateProvider tests into single func

func TestGetProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		provider, err := GetProvider(db, ByName("okta-development"))
		assert.NilError(t, err)
		assert.Assert(t, provider.ID != 0)
		assert.Equal(t, providerDevelop.URL, provider.URL)
	})
}

func TestListProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)
		// TODO: deleted provider
		// TODO: provider in other org
		createProviders(t, db, &providerDevelop, &providerProduction)

		// TODO: test case with default options

		t.Run("exclude infra provider", func(t *testing.T) {
			providers, err := ListProviders(db, ListProvidersOptions{
				ExcludeInfraProvider: true,
			})
			assert.NilError(t, err)
			assert.Equal(t, 2, len(providers))
			// TODO: check ids
		})

		t.Run("by name", func(t *testing.T) {
			providers, err := ListProviders(db, ListProvidersOptions{
				ByName: "okta-development",
			})
			assert.NilError(t, err)
			assert.Equal(t, 1, len(providers))
			// TODO: check ids
		})

		// TODO: test case for ByIDs
		// TODO: test cases for byCreatedBy + NotIDs
		// TODO: test case with pagination
	})
}

func TestUpdateProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			orig := models.Provider{
				Name: "idp",
				Kind: models.ProviderKindGoogle,
			}
			createProviders(t, tx, &orig)

			updated := models.Provider{
				Model:            models.Model{ID: orig.ID},
				Name:             "new-name",
				URL:              "https://example.com/idp",
				ClientID:         "client-id",
				ClientSecret:     "client-secret",
				CreatedBy:        777,
				Kind:             models.ProviderKindAzure,
				AuthURL:          "https://example.com/auth",
				Scopes:           []string{"one", "two"},
				PrivateKey:       "private-key",
				ClientEmail:      "client-email@example.com",
				DomainAdminEmail: "domain-admin-email@example.com",
			}
			err := UpdateProvider(tx, &updated)
			assert.NilError(t, err)

			actual, err := GetProvider(tx, GetProviderOptions{ByID: orig.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, &updated, cmpTimeWithDBPrecision)
		})
		t.Run("name conflict", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			orig := models.Provider{Name: "idp", Kind: models.ProviderKindGoogle}
			other := models.Provider{Name: "taken", Kind: models.ProviderKindOIDC}
			createProviders(t, tx, &orig, &other)

			err := UpdateProvider(tx, &models.Provider{
				Model: models.Model{ID: orig.ID},
				Name:  other.Name,
				Kind:  orig.Kind,
			})

			var ucErr UniqueConstraintError
			assert.Assert(t, errors.As(err, &ucErr))
			expected := UniqueConstraintError{Column: "name", Table: "providers"}
			assert.DeepEqual(t, ucErr, expected)
		})
	})
}

func TestDeleteProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{}
			providerProduction = models.Provider{}
			pu                 = &models.ProviderUser{}
			user               = &models.Identity{}
			i                  = 0
		)

		setup := func() {
			providerDevelop = models.Provider{Name: fmt.Sprintf("okta-development-%d", i), URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: fmt.Sprintf("okta-production-%d", i+1), URL: "prod.okta.com", Kind: models.ProviderKindOkta}
			i += 2

			err := CreateProvider(db, &providerDevelop)
			assert.NilError(t, err)
			err = CreateProvider(db, &providerProduction)
			assert.NilError(t, err)

			providers, err := ListProviders(db, ListProvidersOptions{
				ExcludeInfraProvider: true,
			})
			assert.NilError(t, err)
			assert.Assert(t, len(providers) >= 2)

			user = &models.Identity{Name: "joe@example.com"}
			err = CreateIdentity(db, user)
			assert.NilError(t, err)

			pu, err = CreateProviderUser(db, &providerDevelop, user)
			assert.NilError(t, err)
		}

		t.Run("Deletes work", func(t *testing.T) {
			setup()
			err := DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetProvider(db, ByOptionalName(providerDevelop.Name))
			assert.Error(t, err, "record not found")

			t.Run("provider users are removed", func(t *testing.T) {
				_, err = GetProviderUser(db, pu.ProviderID, pu.IdentityID)
				assert.Error(t, err, "record not found")
			})

			t.Run("user is removed when last providerUser is removed", func(t *testing.T) {
				_, err = GetIdentity(db, GetIdentityOptions{ByID: pu.IdentityID})
				assert.Error(t, err, "record not found")
			})
		})

		t.Run("access keys issued using deleted provider are revoked", func(t *testing.T) {
			setup()

			key := &models.AccessKey{
				Name:       "test key",
				IssuedFor:  user.ID,
				ProviderID: providerDevelop.ID,
				ExpiresAt:  time.Now().Add(5 * time.Minute),
			}

			_, err := CreateAccessKey(db, key)
			assert.NilError(t, err)

			err = DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetAccessKeyByKeyID(db, key.KeyID)
			assert.ErrorContains(t, err, "record not found")
		})

		t.Run("access keys issued using different provider from deleted are NOT revoked", func(t *testing.T) {
			setup()

			_, err := CreateProviderUser(db, &providerProduction, user)
			assert.NilError(t, err)

			key := &models.AccessKey{
				Name:       "test key",
				IssuedFor:  user.ID,
				ProviderID: providerProduction.ID,
				ExpiresAt:  time.Now().Add(5 * time.Minute),
			}

			_, err = CreateAccessKey(db, key)
			assert.NilError(t, err)

			err = DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetAccessKeyByKeyID(db, key.KeyID)
			assert.NilError(t, err)

			// clean up
			err = DeleteProviders(db, DeleteProvidersOptions{ByID: providerProduction.ID})
			assert.NilError(t, err)
		})

		t.Run("user is not removed if there are other providerUsers", func(t *testing.T) {
			setup()

			pu, err := CreateProviderUser(db, &providerProduction, user)
			assert.NilError(t, err)

			err = DeleteProviders(db, DeleteProvidersOptions{ByID: providerDevelop.ID})
			assert.NilError(t, err)

			_, err = GetIdentity(db, GetIdentityOptions{ByID: pu.IdentityID})
			assert.NilError(t, err)
		})
	})
}

func TestCountProvidersByKind(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		createProviders(t, db,
			&models.Provider{Name: "oidc", Kind: "oidc"},
			&models.Provider{Name: "okta", Kind: "okta"},
			&models.Provider{Name: "okta2", Kind: "okta"},
			&models.Provider{Name: "azure", Kind: "azure"},
			&models.Provider{Name: "google", Kind: "google"},
		)

		actual, err := CountProvidersByKind(db)
		assert.NilError(t, err)

		expected := []providersCount{
			{Kind: "azure", Count: 1},
			{Kind: "google", Count: 1},
			{Kind: "oidc", Count: 1},
			{Kind: "okta", Count: 2},
		}

		assert.DeepEqual(t, actual, expected)
	})
}
