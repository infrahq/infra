package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	providerDevelop    = Provider{Kind: "okta", Domain: "dev.okta.com"}
	providerProduction = Provider{Kind: "okta", Domain: "prod.okta.com"}
)

func TestProvider(t *testing.T) {
	db := setup(t)

	err := db.Create(&providerDevelop).Error
	require.NoError(t, err)

	var provider Provider
	err = db.Preload("Okta").First(&provider, &Provider{Kind: "okta"}).Error
	require.NoError(t, err)
	require.Equal(t, ProviderKindOkta, provider.Kind)
	require.Equal(t, "dev.okta.com", provider.Domain)
}

func TestCreateProviderOkta(t *testing.T) {
	db := setup(t)

	provider, err := CreateProvider(db, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func createProviders(t *testing.T, db *gorm.DB, providers ...Provider) {
	for i := range providers {
		_, err := CreateProvider(db, &providers[i])
		require.NoError(t, err)
	}
}

func TestCreateProviderDuplicate(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	_, err := CreateProvider(db, &providerDevelop)
	require.EqualError(t, err, "UNIQUE constraint failed: providers.id")
}

func TestCreateOrUpdateProviderCreate(t *testing.T) {
	db := setup(t)

	provider, err := CreateOrUpdateProvider(db, &providerDevelop, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func TestCreateOrUpdateProviderUpdate(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := CreateOrUpdateProvider(db, &Provider{Domain: "tmp.okta.com"}, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, "tmp.okta.com", provider.Domain)
}

func TestCreateOrUpdateProviderUpdateOkta(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	okta := Provider{
		Kind: ProviderKindOkta,
		Okta: ProviderOkta{
			APIToken: "updated-token",
		},
	}

	provider, err := CreateOrUpdateProvider(db, &okta, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, "updated-token", provider.Okta.APIToken)

	fromDB, err := GetProvider(db, &Provider{Domain: provider.Domain})
	require.NoError(t, err)
	require.Equal(t, "dev.okta.com", fromDB.Domain)
	require.Equal(t, "updated-token", provider.Okta.APIToken)
}

func TestGetProvider(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := GetProvider(db, &Provider{Kind: "okta"})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func TestListProviders(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db, &Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	providers, err = ListProviders(db, &Provider{Kind: "okta", Domain: "dev.okta.com"})
	require.NoError(t, err)
	require.Equal(t, 1, len(providers))
}

func TestDeleteProviders(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db, &Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	err = DeleteProviders(db, &Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.NoError(t, err)

	_, err = GetProvider(db, &Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent provider should not fail
	err = DeleteProviders(db, &Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.NoError(t, err)

	// deleting a provider should not delete unrelated providers
	_, err = GetProvider(db, &Provider{Kind: "okta", Domain: "dev.okta.com"})
	require.NoError(t, err)

	err = DeleteProviders(db, &Provider{Kind: "okta"})
	require.NoError(t, err)

	providers, err = ListProviders(db, &Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 0, len(providers))

	// make sure provider configurations are also being removed
	var okta []ProviderOkta
	err = db.Find(&okta).Error
	require.NoError(t, err)
	require.Equal(t, 0, len(okta))
}
