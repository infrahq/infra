package data

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestProvider(t *testing.T) {
	db := setup(t)

	providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com"}

	err := db.Create(&providerDevelop).Error
	assert.NilError(t, err)

	var provider models.Provider
	err = db.First(&provider, &models.Provider{}).Error
	assert.NilError(t, err)
	assert.Equal(t, "dev.okta.com", provider.URL)
}

func TestCreateProviderOkta(t *testing.T) {
	db := setup(t)

	providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com"}
	err := CreateProvider(db, &providerDevelop)
	assert.NilError(t, err)
	assert.Assert(t, providerDevelop.ID != 0)
	assert.Equal(t, providerDevelop.URL, providerDevelop.URL)
}

func createProviders(t *testing.T, db *gorm.DB, providers ...models.Provider) {
	for i := range providers {
		err := CreateProvider(db, &providers[i])
		assert.NilError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	err := CreateProvider(db, &providerDevelop)
	assert.ErrorIs(t, err, internal.ErrDuplicate)
}

func TestGetProvider(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := GetProvider(db)
	assert.NilError(t, err)
	assert.Assert(t, 0 != provider.ID)
	assert.Equal(t, providerDevelop.URL, provider.URL)
}

func TestListProviders(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(providers))

	providers, err = ListProviders(db, ByURL("dev.okta.com"))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(providers))
}

func TestDeleteProviders(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(providers))

	err = DeleteProviders(db, ByURL("dev.okta.com"))
	assert.NilError(t, err)

	_, err = GetProvider(db, ByURL("dev.okta.com"))
	assert.Error(t, err, "record not found")
}

func TestRecreateProviderSameDomain(t *testing.T) {
	db := setup(t)

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
}
