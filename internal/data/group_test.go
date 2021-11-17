package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	everyone  = Group{Name: "Everyone"}
	engineers = Group{Name: "Engineering"}
	product   = Group{Name: "Product"}
)

func TestGroup(t *testing.T) {
	db := setup(t)

	err := db.Create(&everyone).Error
	require.NoError(t, err)

	var group Group
	err = db.First(&group, &Group{Name: everyone.Name}).Error
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

func createGroups(t *testing.T, db *gorm.DB, groups ...Group) {
	for i := range groups {
		_, err := CreateGroup(db, &groups[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateGroup(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	_, err := CreateGroup(db, &everyone)
	require.EqualError(t, err, "UNIQUE constraint failed: groups.id")
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

	group, err := CreateOrUpdateGroup(db, &Group{Name: "All"}, &everyone)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
	require.Equal(t, "All", group.Name)
}

func TestGetGroup(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	group, err := GetGroup(db, Group{Name: everyone.Name})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, group.ID)
}

func TestListGroups(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	groups, err := ListGroups(db, &Group{})
	require.NoError(t, err)
	require.Equal(t, 3, len(groups))

	groups, err = ListGroups(db, &Group{Name: engineers.Name})
	require.NoError(t, err)
	require.Equal(t, 1, len(groups))
}

func TestGroupBindRoles(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateRole(db, &admin)
	require.NoError(t, err)

	groups, err := ListGroups(db, &Group{})
	require.NoError(t, err)

	for _, group := range groups {
		err = group.BindRoles(db, admin.ID)
		require.NoError(t, err)
	}

	roles, err := ListRoles(db, &Role{})
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Len(t, roles[0].Groups, 3)
	require.ElementsMatch(t, []string{
		everyone.Name, engineers.Name, product.Name,
	}, []string{
		roles[0].Groups[0].Name,
		roles[0].Groups[1].Name,
		roles[0].Groups[2].Name,
	})
}

func TestGroupBindMoreRoles(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateRole(db, &admin)
	require.NoError(t, err)

	group, err := GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	err = group.BindRoles(db, admin.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Roles, 1)

	view := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "view",
		},
	}

	_, err = CreateRole(db, &view)
	require.NoError(t, err)

	err = group.BindRoles(db, admin.ID, view.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Roles, 2)
}

func TestGroupBindLessRoles(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	admin := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "admin",
		},
	}

	view := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "view",
		},
	}

	_, err := CreateRole(db, &admin)
	require.NoError(t, err)

	_, err = CreateRole(db, &view)
	require.NoError(t, err)

	group, err := GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	err = group.BindRoles(db, admin.ID, view.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Roles, 2)

	err = group.BindRoles(db, admin.ID)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Roles, 1)
}

func TestGroupBindUsers(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	bond, err := CreateUser(db, &bond)
	require.NoError(t, err)

	groups, err := ListGroups(db, &Group{})
	require.NoError(t, err)

	for _, group := range groups {
		err := group.BindUsers(db, *bond)
		require.NoError(t, err)
	}

	user, err := GetUser(db, &User{Email: bond.Email})
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

	group, err := GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	err = group.BindUsers(db, *bond)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 1)

	bourne, err := CreateUser(db, &bourne)
	require.NoError(t, err)

	err = group.BindUsers(db, *bond, *bourne)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
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

	group, err := GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	err = group.BindUsers(db, *bourne, *bauer)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 2)

	err = group.BindUsers(db, *bauer)
	require.NoError(t, err)

	group, err = GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)
	require.Len(t, group.Users, 1)
}

func TestDeleteGroup(t *testing.T) {
	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	_, err := GetGroup(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	err = DeleteGroups(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	_, err = GetGroup(db, &Group{Name: everyone.Name})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent group should not fail
	err = DeleteGroups(db, &Group{Name: everyone.Name})
	require.NoError(t, err)

	// deleting an group should not delete unrelated groups
	_, err = GetGroup(db, &Group{Name: engineers.Name})
	require.NoError(t, err)
}
