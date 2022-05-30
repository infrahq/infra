package data

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestProvider(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {

		providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com"}

		err := db.Create(&providerDevelop).Error
		assert.NilError(t, err)

		var provider models.Provider
		err = db.Not("name = ?", models.InternalInfraProviderName).First(&provider).Error
		assert.NilError(t, err)
		assert.Equal(t, "dev.okta.com", provider.URL)
	})
}

func createProviders(t *testing.T, db *gorm.DB, providers ...models.Provider) {
	for i := range providers {
		err := CreateProvider(db, &providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		provider, err := GetProvider(db, ByName("okta-development"))
		assert.NilError(t, err)
		assert.Assert(t, 0 != provider.ID)
		assert.Equal(t, providerDevelop.URL, provider.URL)
	})
}

func TestListProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		providers, err := ListProviders(db, NotName(models.InternalInfraProviderName))
		assert.NilError(t, err)
		assert.Equal(t, 2, len(providers))

		providers, err = ListProviders(db, ByOptionalName("okta-development"))
		assert.NilError(t, err)
		assert.Equal(t, 1, len(providers))
	})
}

func TestDeleteProviders(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		providers, err := ListProviders(db, NotName(models.InternalInfraProviderName))
		assert.NilError(t, err)
		assert.Equal(t, 2, len(providers))

		err = DeleteProviders(db, ByOptionalName("okta-development"))
		assert.NilError(t, err)

		_, err = GetProvider(db, ByOptionalName("okta-development"))
		assert.Error(t, err, "record not found")
	})
}

func TestRecreateProviderSameDomain(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
			providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
		)

		createProviders(t, db, providerDevelop, providerProduction)

		err := DeleteProviders(db, func(db *gorm.DB) *gorm.DB {
			return db.Where(&models.Provider{Name: "okta-development", URL: "dev.okta.com"})
		})
		assert.NilError(t, err)

		err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "dev.okta.com"})
		assert.NilError(t, err)
	})
}
