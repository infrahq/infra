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

func TestProviderSetUsers(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	providers, err := ListProviders(db, &Provider{})
	require.NoError(t, err)

	for _, provider := range providers {
		err := provider.SetUsers(db, bond.Email)
		require.NoError(t, err)
	}

	users, err := ListUsers(db, &User{})
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

	provider, err := GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 0)

	err = provider.SetUsers(db, bond.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 1)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	err = provider.SetUsers(db, bond.Email, bourne.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
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

	provider, err := GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 0)

	err = provider.SetUsers(db, bourne.Email, bauer.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 2)

	err = provider.SetUsers(db, bauer.Email)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Users, 1)
}

func TestProviderSetGroups(t *testing.T) {
	db := setup(t)
	createProviders(t, db, providerDevelop, providerProduction)

	everyone, err := CreateGroup(db, &everyone)
	require.NoError(t, err)

	providers, err := ListProviders(db, &Provider{})
	require.NoError(t, err)

	for _, provider := range providers {
		err := provider.SetGroups(db, everyone.Name)
		require.NoError(t, err)
	}

	groups, err := ListGroups(db, &Group{})
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

	provider, err := GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 0)

	err = provider.SetGroups(db, everyone.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 1)

	engineers, err := CreateGroup(db, &engineers)
	require.NoError(t, err)

	err = provider.SetGroups(db, everyone.Name, engineers.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
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

	provider, err := GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 0)

	err = provider.SetGroups(db, engineers.Name, product.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 2)

	err = provider.SetGroups(db, product.Name)
	require.NoError(t, err)

	provider, err = GetProvider(db, &Provider{Domain: providerDevelop.Domain})
	require.NoError(t, err)
	require.Len(t, provider.Groups, 1)
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
