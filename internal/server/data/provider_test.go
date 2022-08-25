package data

import (
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestProvider(t *testing.T) {
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

func createProviders(t *testing.T, db GormTxn, providers ...models.Provider) {
	for i := range providers {
		err := CreateProvider(db, &providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		err := CreateProvider(db, &providerDevelop)

		var uniqueConstraintErr UniqueConstraintError
		assert.Assert(t, errors.As(err, &uniqueConstraintErr), "error is wrong type %T", err)
		expected := UniqueConstraintError{Column: "name", Table: "providers"}
		assert.DeepEqual(t, uniqueConstraintErr, expected)
	})
}

func TestGetProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		provider, err := GetProvider(db, ByName("okta-development"))
		assert.NilError(t, err)
		assert.Assert(t, 0 != provider.ID)
		assert.Equal(t, providerDevelop.URL, provider.URL)
	})
}

func TestListProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		providers, err := ListProviders(db, nil, NotName(models.InternalInfraProviderName))
		assert.NilError(t, err)
		assert.Equal(t, 2, len(providers))

		providers, err = ListProviders(db, nil, ByOptionalName("okta-development"))
		assert.NilError(t, err)
		assert.Equal(t, 1, len(providers))
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

			providers, err := ListProviders(db, nil, NotName(models.InternalInfraProviderName))
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
			err := DeleteProviders(db, ByOptionalName(providerDevelop.Name))
			assert.NilError(t, err)

			_, err = GetProvider(db, ByOptionalName(providerDevelop.Name))
			assert.Error(t, err, "record not found")

			t.Run("provider users are removed", func(t *testing.T) {
				_, err = GetProviderUser(db, pu.ProviderID, pu.IdentityID)
				assert.Error(t, err, "record not found")
			})

			t.Run("user is removed when last providerUser is removed", func(t *testing.T) {
				_, err = GetIdentity(db, ByID(pu.IdentityID))
				assert.Error(t, err, "record not found")
			})
		})

		t.Run("user is not removed if there are other providerUsers", func(t *testing.T) {
			setup()

			pu, err := CreateProviderUser(db, &providerProduction, user)
			assert.NilError(t, err)

			err = DeleteProviders(db, ByOptionalName(providerDevelop.Name))
			assert.NilError(t, err)

			_, err = GetIdentity(db, ByID(pu.IdentityID))
			assert.NilError(t, err)
		})
	})
}

func TestRecreateProviderSameDomain(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		err := DeleteProviders(db, func(db *gorm.DB) *gorm.DB {
			return db.Where(&models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta})
		})
		assert.NilError(t, err)

		err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "example.com", Kind: models.ProviderKindOkta})
		assert.NilError(t, err)
	})
}

func TestCountProvidersByKind(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		assert.NilError(t, CreateProvider(db, &models.Provider{Name: "oidc", Kind: "oidc"}))
		assert.NilError(t, CreateProvider(db, &models.Provider{Name: "okta", Kind: "okta"}))
		assert.NilError(t, CreateProvider(db, &models.Provider{Name: "okta2", Kind: "okta"}))
		assert.NilError(t, CreateProvider(db, &models.Provider{Name: "azure", Kind: "azure"}))
		assert.NilError(t, CreateProvider(db, &models.Provider{Name: "google", Kind: "google"}))

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
