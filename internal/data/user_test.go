package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	bond   = User{Email: "jbond@infrahq.com"}
	bourne = User{Email: "jbourne@infrahq.com"}
	bauer  = User{Email: "jbauer@infrahq.com"}
)

func TestUser(t *testing.T) {
	db := setup(t)

	err := db.Create(&bond).Error
	require.NoError(t, err)

	var user User
	err = db.First(&user, &User{Email: bond.Email}).Error
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, bond.Email, user.Email)
}

func TestCreateUser(t *testing.T) {
	db := setup(t)

	user, err := CreateUser(db, &bond)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, bond.Email, user.Email)
}

func createUsers(t *testing.T, db *gorm.DB, users ...User) {
	for i := range users {
		_, err := CreateUser(db, &users[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	_, err := CreateUser(db, &bond)
	require.EqualError(t, err, "UNIQUE constraint failed: users.id")
}

func TestCreateOrUpdateUserCreate(t *testing.T) {
	db := setup(t)

	user, err := CreateOrUpdateUser(db, &bond, &bond)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, bond.Email, user.Email)
}

func TestCreateOrUpdateUserUpdate(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	user, err := CreateOrUpdateUser(db, &User{Email: "james@infrahq.com"}, &bond)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, "james@infrahq.com", user.Email)
}

func TestGetUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	user, err := GetUser(db, User{Email: bond.Email})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
}

func TestListUsers(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	users, err := ListUsers(db, &User{})
	require.NoError(t, err)
	require.Equal(t, 3, len(users))

	users, err = ListUsers(db, &User{Email: bourne.Email})
	require.NoError(t, err)
	require.Equal(t, 1, len(users))
}

func TestUserBindRoles(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	admin := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateRole(db, &admin)
	require.NoError(t, err)

	users, err := ListUsers(db, &User{})
	require.NoError(t, err)

	for _, user := range users {
		err := user.BindRoles(db, admin.ID)
		require.NoError(t, err)
	}

	roles, err := ListRoles(db, &Role{})
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Len(t, roles[0].Users, 3)
	require.ElementsMatch(t, []string{
		bond.Email, bourne.Email, bauer.Email,
	}, []string{
		roles[0].Users[0].Email,
		roles[0].Users[1].Email,
		roles[0].Users[2].Email,
	})
}

func TestUserBindMoreRoles(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	admin := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateRole(db, &admin)
	require.NoError(t, err)

	user, err := GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 0)

	err = user.BindRoles(db, admin.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 1)

	view := Role{
		Kind: RoleKindKubernetes,
		Kubernetes: RoleKubernetes{
			Kind: RoleKubernetesKindRole,
			Name: "view",
		},
	}

	_, err = CreateRole(db, &view)
	require.NoError(t, err)

	err = user.BindRoles(db, admin.ID, view.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 2)
}

func TestUserBindLessRoles(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

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

	user, err := GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 0)

	err = user.BindRoles(db, admin.ID, view.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 2)

	err = user.BindRoles(db, admin.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Roles, 1)
}

func TestDeleteUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	_, err := GetUser(db, &User{Email: bond.Email})
	require.NoError(t, err)

	err = DeleteUsers(db, &User{Email: bond.Email})
	require.NoError(t, err)

	_, err = GetUser(db, &User{Email: bond.Email})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent user should not fail
	err = DeleteUsers(db, &User{Email: bond.Email})
	require.NoError(t, err)

	// deleting an user should not delete unrelated users
	_, err = GetUser(db, &User{Email: bourne.Email})
	require.NoError(t, err)
}
