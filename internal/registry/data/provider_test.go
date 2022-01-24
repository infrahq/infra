package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

var (
	providerDevelop    = models.Provider{Kind: "okta", Domain: "dev.okta.com"}
	providerProduction = models.Provider{Kind: "okta", Domain: "prod.okta.com"}
)

func TestProvider(t *testing.T) {
	db := setup(t)

	err := db.Create(&providerDevelop).Error
	require.NoError(t, err)

	var provider models.Provider
	err = db.First(&provider, &models.Provider{Kind: "okta"}).Error
	require.NoError(t, err)
	require.Equal(t, models.ProviderKindOkta, provider.Kind)
	require.Equal(t, "dev.okta.com", provider.Domain)
}

func TestCreateProviderOkta(t *testing.T) {
	db := setup(t)

	provider, err := CreateProvider(db, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, 0, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func createProviders(t *testing.T, db *gorm.DB, providers ...models.Provider) {
	for i := range providers {
		_, err := CreateProvider(db, &providers[i])
		require.NoError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	_, err := CreateProvider(db, &providerDevelop)
	require.EqualError(t, err, "duplicate record")
}

func TestCreateOrUpdateProviderCreate(t *testing.T) {
	db := setup(t)

	provider, err := CreateOrUpdateProvider(db, &providerDevelop, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, 0, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func TestCreateOrUpdateProviderUpdate(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := CreateOrUpdateProvider(db, &models.Provider{Domain: "tmp.okta.com"}, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, 0, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, "tmp.okta.com", provider.Domain)
}

func TestGetProvider(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := GetProvider(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.NotEqual(t, 0, provider.ID)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func TestListProviders(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	providers, err = ListProviders(db, &models.Provider{Kind: "okta", Domain: "dev.okta.com"})
	require.NoError(t, err)
	require.Equal(t, 1, len(providers))
}

func TestDeleteProviders(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	byKindAndDomain := func(kind, domain string) SelectorFunc {
		return func(db *gorm.DB) *gorm.DB {
			if kind != "" {
				db = db.Where("kind = ?", kind)
			}

			if domain != "" {
				db = db.Where("domain = ?", domain)
			}

			return db
		}
	}

	err = DeleteProviders(db, byKindAndDomain("okta", "prod.okta.com"))
	require.NoError(t, err)

	_, err = GetProvider(db, &models.Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent provider should return NotFound
	err = DeleteProviders(db, byKindAndDomain("okta", "prod.okta.com"))
	require.EqualError(t, err, "record not found")

	// deleting a provider should not delete unrelated providers
	_, err = GetProvider(db, &models.Provider{Kind: "okta", Domain: "dev.okta.com"})
	require.NoError(t, err)

	err = DeleteProviders(db, byKindAndDomain("okta", ""))
	require.NoError(t, err)

	providers, err = ListProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 0, len(providers))
}

func TestRecreateProviderSameDomain(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	err := DeleteProviders(db, func(db *gorm.DB) *gorm.DB {
		return db.Where(&models.Provider{Domain: "dev.okta.com"})
	})
	require.NoError(t, err)

	_, err = CreateProvider(db, &models.Provider{Domain: "dev.okta.com"})
	require.NoError(t, err)
}
