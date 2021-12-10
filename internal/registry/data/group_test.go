package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

var (
	everyone  = models.Group{Name: "Everyone"}
	engineers = models.Group{Name: "Engineering"}
	product   = models.Group{Name: "Product"}
)

func TestGroup(t *testing.T) {
	db := setup(t)

	err := db.Create(&everyone).Error
	require.NoError(t, err)

	var group models.Group
	err = db.First(&group, &models.Group{Name: everyone.Name}).Error
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
	require.Equal(t, everyone.Name, group.Name)
}

func TestCreateGroup(t *testing.T) {
	db := setup(t)

	group, err := CreateGroup(db, &everyone)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
	require.Equal(t, everyone.Name, group.Name)
}

func createGroups(t *testing.T, db *gorm.DB, groups ...models.Group) {
	for i := range groups {
		_, err := CreateGroup(db, &groups[i])
		require.NoError(t, err)
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	_, err := CreateGroup(db, &everyone)
	require.EqualError(t, err, "duplicate record")
}

func TestCreateOrUpdateGroupCreate(t *testing.T) {
	db := setup(t)

	group, err := CreateOrUpdateGroup(db, &everyone, &everyone)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
	require.Equal(t, everyone.Name, group.Name)
}

func TestCreateOrUpdateGroupUpdate(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	group, err := CreateOrUpdateGroup(db, &models.Group{Name: "All"}, &everyone)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
	require.Equal(t, "All", group.Name)
}

func TestGetGroup(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	group, err := GetGroup(db, models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
}

func TestListGroups(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	groups, err := ListGroups(db, &models.Group{})
	require.NoError(t, err)
	require.Equal(t, 3, len(groups))

	groups, err = ListGroups(db, &models.Group{Name: engineers.Name})
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))
}

func TestGroupBindGrants(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateGrant(db, &admin)
	require.NoError(t, err)

	groups, err := ListGroups(db, &models.Group{})
	require.NoError(t, err)

	for i := range groups {
		err := BindGroupGrants(db, &groups[i], admin.ID)
		require.NoError(t, err)
	}

	grants, err := ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Len(t, grants[0].Groups, 3)
	require.ElementsMatch(t, []string{
		everyone.Name, engineers.Name, product.Name,
	}, []string{
		grants[0].Groups[0].Name,
		grants[0].Groups[1].Name,
		grants[0].Groups[2].Name,
	})
}

func TestGroupBindMoreGrants(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateGrant(db, &admin)
	require.NoError(t, err)

	group, err := GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 0)

	err = BindGroupGrants(db, group, admin.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 1)

	view := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "view",
		},
	}

	_, err = CreateGrant(db, &view)
	require.NoError(t, err)

	err = BindGroupGrants(db, group, admin.ID, view.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 2)
}

func TestGroupBindLessGrants(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "admin",
		},
	}

	view := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "view",
		},
	}

	_, err := CreateGrant(db, &admin)
	require.NoError(t, err)

	_, err = CreateGrant(db, &view)
	require.NoError(t, err)

	group, err := GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 0)

	err = BindGroupGrants(db, group, admin.ID, view.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 2)

	err = BindGroupGrants(db, group, admin.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Grants, 1)
}

func TestGroupBindUsers(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	groups, err := ListGroups(db, &models.Group{})
	require.NoError(t, err)

	for i := range groups {
		err := BindGroupUsers(db, &groups[i], *bond)
		require.NoError(t, err)
	}

	user, err := GetUser(db, &models.User{Email: bond.Email})
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
	createGroups(t, db, everyone, engineers, product)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	group, err := GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 0)

	err = BindGroupUsers(db, group, *bond)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 1)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	err = BindGroupUsers(db, group, *bond, *bourne)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 2)
}

func TestGroupBindLessUsers(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	bauer, err := CreateUser(db, &bauer)
	require.NoError(t, err)

	group, err := GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 0)

	err = BindGroupUsers(db, group, *bourne, *bauer)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 2)

	err = BindGroupUsers(db, group, *bauer)
	require.NoError(t, err)

	group, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 1)
}

func TestDeleteGroup(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	_, err := GetGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)

	err = DeleteGroups(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)

	_, err = GetGroup(db, &models.Group{Name: everyone.Name})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent group should not fail
	err = DeleteGroups(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)

	// deleting an group should not delete unrelated groups
	_, err = GetGroup(db, &models.Group{Name: engineers.Name})
	require.NoError(t, err)
}

func TestRecreateGroupSameName(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	err := DeleteGroups(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)

	_, err = CreateGroup(db, &models.Group{Name: everyone.Name})
	require.NoError(t, err)
}
