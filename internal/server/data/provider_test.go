package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestProvider(t *testing.T) {
	db := setup(t)

	providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com"}

	err := db.Create(&providerDevelop).Error
	require.NoError(t, err)

	var provider models.Provider
	err = db.First(&provider, &models.Provider{}).Error
	require.NoError(t, err)
	require.Equal(t, "dev.okta.com", provider.URL)
}

func TestCreateProviderOkta(t *testing.T) {
	db := setup(t)

	providerDevelop := models.Provider{Name: "okta-development", URL: "dev.okta.com"}

	p := providerDevelop
	err := CreateProvider(db, &p)
	require.NoError(t, err)
	require.NotEqual(t, 0, providerDevelop.ID)
	require.Equal(t, providerDevelop.URL, providerDevelop.URL)
}

func createProviders(t *testing.T, db *gorm.DB, providers ...models.Provider) {
	for i := range providers {
		err := CreateProvider(db, &providers[i])
		require.NoError(t, err)
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
	require.ErrorIs(t, err, internal.ErrDuplicate)
}

func TestGetProvider(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := GetProvider(db)
	require.NoError(t, err)
	require.NotEqual(t, 0, provider.ID)
	require.Equal(t, providerDevelop.URL, provider.URL)
}

func TestListProviders(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db)
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	providers, err = ListProviders(db, ByURL("dev.okta.com"))
	require.NoError(t, err)
	require.Equal(t, 1, len(providers))
}

func TestDeleteProviders(t *testing.T) {
	db := setup(t)

	var (
		providerDevelop    = models.Provider{Name: "okta-development", URL: "dev.okta.com"}
		providerProduction = models.Provider{Name: "okta-production", URL: "prod.okta.com"}
	)

	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db)
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	err = DeleteProviders(db, ByURL("dev.okta.com"))
	require.NoError(t, err)

	_, err = GetProvider(db, ByURL("dev.okta.com"))
	require.EqualError(t, err, "record not found")
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
	require.NoError(t, err)

	err = CreateProvider(db, &models.Provider{Name: "okta-development", URL: "dev.okta.com"})
	require.NoError(t, err)
}
