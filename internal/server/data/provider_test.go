package data

import (
	"errors"
	"fmt"
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: combine this test case with TestListProvider
func TestProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta}

		err := db.Create(&providerDevelop).Error
		assert.NilError(t, err)

		var provider models.Provider
		err = db.Not("name = ?", models.InternalInfraProviderName).First(&provider).Error
		assert.NilError(t, err)
		assert.Equal(t, "dev.okta.com", provider.URL)
	})
}

func createProviders(t *testing.T, db *gorm.DB, providers ...*models.Provider) {
	for i := range providers {
		err := CreateProvider(db, providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		providerDevelop.ID = 0
		err := CreateProvider(db, &providerDevelop)

		var uniqueConstraintErr UniqueConstraintError
		assert.Assert(t, errors.As(err, &uniqueConstraintErr), "error is wrong type %T", err)
		expected := UniqueConstraintError{Column: "name", Table: "providers"}
		assert.DeepEqual(t, uniqueConstraintErr, expected)
	})
}

func TestGetProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		t.Run("by ID", func(t *testing.T) {
			provider, err := GetProvider(db, ByIDQ(providerProduction.ID))
			assert.NilError(t, err)
			assert.Equal(t, providerProduction.ID, provider.ID)
			assert.Equal(t, providerProduction.URL, provider.URL)
		})

		t.Run("by name", func(t *testing.T) {
			provider, err := GetProvider(db, ByNameQ("okta-development"))
			assert.NilError(t, err)
			assert.Equal(t, providerDevelop.ID, provider.ID)
			assert.Equal(t, providerDevelop.URL, provider.URL)
		})

		t.Run("does not exist", func(t *testing.T) {
			_, err := GetProvider(db, ByNameQ("does-not-exist"))
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

	})
}

func TestListProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		providers, err := ListProviders(db, &models.Pagination{}, NotName(models.InternalInfraProviderName))
		assert.NilError(t, err)
		assert.Equal(t, 2, len(providers))

		providers, err = ListProviders(db, &models.Pagination{}, ByOptionalName("okta-development"))
		assert.NilError(t, err)
		assert.Equal(t, 1, len(providers))
	})
}

func TestDeleteProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{}
			providerProduction = models.Provider{}
			pu                 = &models.ProviderUser{}
			user               = &models.Identity{}
			i                  = 0
		)
		setup := func() {
			providerDevelop = models.Provider{Name: fmt.Sprintf("okta-development-%d", i), URL: "dev.okta.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: fmt.Sprintf("okta-production-%d", i+1), URL: "prod.okta.com", Kind: models.ProviderKindOkta}
			i += 2

			err := CreateProvider(db, &providerDevelop)
			assert.NilError(t, err)
			err = CreateProvider(db, &providerProduction)
			assert.NilError(t, err)

			providers, err := ListProviders(db, &models.Pagination{}, NotName(models.InternalInfraProviderName))
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

			_, err = GetProvider(db, ByNameQ(providerDevelop.Name))
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com", Kind: models.ProviderKindOkta}
		)

		createProviders(t, db, &providerDevelop, &providerProduction)

		err := DeleteProviders(db, func(db *gorm.DB) *gorm.DB {
			return db.Where(&models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta})
		})
		assert.NilError(t, err)

		err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "dev.okta.com", Kind: models.ProviderKindOkta})
		assert.NilError(t, err)
	})
}

func TestCountProvidersByKind(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
			{Kind: "infra", Count: 1},
			{Kind: "oidc", Count: 1},
			{Kind: "okta", Count: 2},
		}

		assert.DeepEqual(t, actual, expected)
	})
}
