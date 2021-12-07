package data

import (
	"testing"

	"github.com/google/uuid"
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
	err = db.Preload("Okta").First(&provider, &models.Provider{Kind: "okta"}).Error
	require.NoError(t, err)
	require.Equal(t, models.ProviderKindOkta, provider.Kind)
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
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, providerDevelop.Domain, provider.Domain)
}

func TestCreateOrUpdateProviderUpdate(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := CreateOrUpdateProvider(db, &models.Provider{Domain: "tmp.okta.com"}, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.Equal(t, providerDevelop.Kind, provider.Kind)
	require.Equal(t, "tmp.okta.com", provider.Domain)
}

func TestCreateOrUpdateProviderUpdateOkta(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	okta := models.Provider{
		Kind: models.ProviderKindOkta,
		Okta: models.ProviderOkta{
			APIToken: "updated-token",
		},
	}

	provider, err := CreateOrUpdateProvider(db, &okta, &providerDevelop)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
	require.EqualValues(t, "updated-token", provider.Okta.APIToken)

	fromDB, err := GetProvider(db, &models.Provider{Domain: provider.Domain})
	require.NoError(t, err)
	require.Equal(t, "dev.okta.com", fromDB.Domain)
	require.EqualValues(t, "updated-token", provider.Okta.APIToken)
}

func TestGetProvider(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	provider, err := GetProvider(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, provider.ID)
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

func TestProviderSetUsers(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	providers, err := ListProviders(db, &models.Provider{})
	require.NoError(t, err)

	for i := range providers {
		err := SetProviderUsers(db, &providers[i], bond.Email)
		require.NoError(t, err)
	}

	users, err := ListUsers(db, &models.User{})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Len(t, users[0].Providers, 2)
	require.ElementsMatch(t, []string{
		"dev.okta.com", "prod.okta.com",
	}, []string{
		users[0].Providers[0].Domain,
		users[0].Providers[1].Domain,
	})
}

func TestProviderSetMoreUsers(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	provider, err := GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 0)

	err = SetProviderUsers(db, provider, bond.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 1)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	err = SetProviderUsers(db, provider, bond.Email, bourne.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 2)
}

func TestProviderSetLessUsers(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	bauer, err := CreateUser(db, &bauer)
	require.NoError(t, err)

	provider, err := GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 0)

	err = SetProviderUsers(db, provider, bourne.Email, bauer.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 2)

	err = SetProviderUsers(db, provider, bauer.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 1)
}

func TestProviderSetGroups(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	everyone, err := CreateGroup(db, &everyone)
	require.NoError(t, err)

	providers, err := ListProviders(db, &models.Provider{})
	require.NoError(t, err)

	for i := range providers {
		err := SetProviderGroups(db, &providers[i], everyone.Name)
		require.NoError(t, err)
	}

	groups, err := ListGroups(db, &models.Group{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Providers, 2)
	require.ElementsMatch(t, []string{
		"dev.okta.com", "prod.okta.com",
	}, []string{
		groups[0].Providers[0].Domain,
		groups[0].Providers[1].Domain,
	})
}

func TestProviderSetMoreGroups(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	everyone, err := CreateGroup(db, &everyone)
	require.NoError(t, err)

	provider, err := GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 0)

	err = SetProviderGroups(db, provider, everyone.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 1)

	engineers, err := CreateGroup(db, &engineers)
	require.NoError(t, err)

	err = SetProviderGroups(db, provider, everyone.Name, engineers.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 2)
}

func TestProviderSetLessGroups(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	engineers, err := CreateGroup(db, &engineers)
	require.NoError(t, err)

	product, err := CreateGroup(db, &product)
	require.NoError(t, err)

	provider, err := GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 0)

	err = SetProviderGroups(db, provider, engineers.Name, product.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 2)

	err = SetProviderGroups(db, provider, product.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &models.Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 1)
}

func TestDeleteProviders(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	providers, err := ListProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 2, len(providers))

	err = DeleteProviders(db, &models.Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.NoError(t, err)

	_, err = GetProvider(db, &models.Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent provider should not fail
	err = DeleteProviders(db, &models.Provider{Kind: "okta", Domain: "prod.okta.com"})
	require.NoError(t, err)

	// deleting a provider should not delete unrelated providers
	_, err = GetProvider(db, &models.Provider{Kind: "okta", Domain: "dev.okta.com"})
	require.NoError(t, err)

	err = DeleteProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)

	providers, err = ListProviders(db, &models.Provider{Kind: "okta"})
	require.NoError(t, err)
	require.Equal(t, 0, len(providers))

	// make sure provider configurations are also being removed
	var okta []models.ProviderOkta
	err = db.Find(&okta).Error
	require.NoError(t, err)
	require.Equal(t, 0, len(okta))
}
