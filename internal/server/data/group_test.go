package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestGroup(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone = models.Group{Name: "Everyone", ProviderID: providerID}
	)

	err := db.Create(&everyone).Error
	require.NoError(t, err)

	var group models.Group
	err = db.First(&group, &models.Group{Name: everyone.Name}).Error
	require.NoError(t, err)
	require.NotEqual(t, 0, group.ID)
	require.Equal(t, everyone.Name, group.Name)
}

func TestCreateGroup(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone = models.Group{Name: "Everyone", ProviderID: providerID}
	)

	err := CreateGroup(db, &everyone)
	group := everyone
	require.NoError(t, err)
	require.NotEqual(t, 0, group.ID)
	require.Equal(t, everyone.Name, group.Name)
}

func createGroups(t *testing.T, db *gorm.DB, groups ...models.Group) {
	for i := range groups {
		err := CreateGroup(db, &groups[i])
		require.NoError(t, err)
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	err := CreateGroup(db, &models.Group{Name: "Everyone", ProviderID: providerID})
	require.Contains(t, err.Error(), "duplicate record")
}

func TestGetGroup(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	group, err := GetGroup(db, ByName(everyone.Name))
	require.NoError(t, err)
	require.NotEqual(t, 0, group.ID)
}

func TestListGroups(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	groups, err := ListGroups(db)
	require.NoError(t, err)
	require.Equal(t, 3, len(groups))

	groups, err = ListGroups(db, ByName(engineers.Name))
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))
}

func TestGroupBindUsers(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bond      = models.User{Email: "jbond@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateUser(db, &bond)
	require.NoError(t, err)

	groups, err := ListGroups(db)
	require.NoError(t, err)

	for i := range groups {
		err := BindGroupUsers(db, &groups[i], bond)
		require.NoError(t, err)
	}

	user, err := GetUser(db.Preload("Groups"), ByEmail(bond.Email))
	require.NoError(t, err)
	require.Len(t, user.Groups, 3)
	require.ElementsMatch(t, []string{
		everyone.Name, engineers.Name, product.Name,
	}, []string{
		user.Groups[0].Name,
		user.Groups[1].Name,
		user.Groups[2].Name,
	})
}

func TestGroupBindMoreUsers(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bond      = models.User{Email: "jbond@infrahq.com", ProviderID: providerID}
		bourne    = models.User{Email: "jbourne@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateUser(db, &bond)
	require.NoError(t, err)

	group, err := GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 0)

	err = BindGroupUsers(db, group, bond)
	require.NoError(t, err)

	group, err = GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 1)

	err = CreateUser(db, &bourne)
	require.NoError(t, err)

	err = BindGroupUsers(db, group, bond, bourne)
	require.NoError(t, err)

	group, err = GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 2)
}

func TestGroupBindLessUsers(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bourne    = models.User{Email: "jbourne@infrahq.com", ProviderID: providerID}
		bauer     = models.User{Email: "jbauer@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateUser(db, &bourne)
	require.NoError(t, err)

	err = CreateUser(db, &bauer)
	require.NoError(t, err)

	group, err := GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 0)

	err = BindGroupUsers(db, group, bourne, bauer)
	require.NoError(t, err)

	group, err = GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 2)

	err = BindGroupUsers(db, group, bauer)
	require.NoError(t, err)

	group, err = GetGroup(db.Preload("Users"), ByName(everyone.Name))
	require.NoError(t, err)
	require.Len(t, group.Users, 1)
}

func TestDeleteGroup(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	_, err := GetGroup(db, ByName(everyone.Name))
	require.NoError(t, err)

	err = DeleteGroups(db, ByName(everyone.Name))
	require.NoError(t, err)

	_, err = GetGroup(db, ByName(everyone.Name))
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent group should not fail
	err = DeleteGroups(db, ByName(everyone.Name))
	require.NoError(t, err)

	// deleting an group should not delete unrelated groups
	_, err = GetGroup(db, ByName(engineers.Name))
	require.NoError(t, err)
}

func TestRecreateGroupSameName(t *testing.T) {
	db := setup(t)

	var providerID = uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := DeleteGroups(db, ByName(everyone.Name))
	require.NoError(t, err)

	err = CreateGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
}
