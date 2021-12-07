package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

var (
	bond   = models.User{Email: "jbond@infrahq.com"}
	bourne = models.User{Email: "jbourne@infrahq.com"}
	bauer  = models.User{Email: "jbauer@infrahq.com"}
)

func TestUser(t *testing.T) {
	db := setup(t)

	err := db.Create(&bond).Error
	require.NoError(t, err)

	var user models.User
	err = db.First(&user, &models.User{Email: bond.Email}).Error
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

func createUsers(t *testing.T, db *gorm.DB, users ...models.User) {
	for i := range users {
		_, err := CreateUser(db, &users[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	_, err := CreateUser(db, &bond)
	require.EqualError(t, err, "duplicate record")
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

	user, err := CreateOrUpdateUser(db, &models.User{Email: "james@infrahq.com"}, &bond)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
	require.Equal(t, "james@infrahq.com", user.Email)
}

func TestGetUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	user, err := GetUser(db, models.User{Email: bond.Email})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, user.ID)
}

func TestListUsers(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	users, err := ListUsers(db, &models.User{})
	require.NoError(t, err)
	require.Equal(t, 3, len(users))

	users, err = ListUsers(db, &models.User{Email: bourne.Email})
	require.NoError(t, err)
	require.Equal(t, 1, len(users))
}

func TestUserBindGrants(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	admin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateGrant(db, &admin)
	require.NoError(t, err)

	users, err := ListUsers(db, &models.User{})
	require.NoError(t, err)

	for i := range users {
		err := BindUserGrants(db, &users[i], admin.ID)
		require.NoError(t, err)
	}

	grants, err := ListGrants(db, &models.Grant{})
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Len(t, grants[0].Users, 3)
	require.ElementsMatch(t, []string{
		bond.Email, bourne.Email, bauer.Email,
	}, []string{
		grants[0].Users[0].Email,
		grants[0].Users[1].Email,
		grants[0].Users[2].Email,
	})
}

func TestUserBindMoreGrants(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	admin := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "admin",
		},
	}

	_, err := CreateGrant(db, &admin)
	require.NoError(t, err)

	user, err := GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 0)

	err = BindUserGrants(db, user, admin.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 1)

	view := models.Grant{
		Kind: models.GrantKindKubernetes,
		Kubernetes: models.GrantKubernetes{
			Kind: models.GrantKubernetesKindRole,
			Name: "view",
		},
	}

	_, err = CreateGrant(db, &view)
	require.NoError(t, err)

	err = BindUserGrants(db, user, admin.ID, view.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 2)
}

func TestUserBindLessGrants(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

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

	user, err := GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 0)

	err = BindUserGrants(db, user, admin.ID, view.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 2)

	err = BindUserGrants(db, user, admin.ID)
	require.NoError(t, err)

	user, err = GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
	require.Len(t, user.Grants, 1)
}

func TestDeleteUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	_, err := GetUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)

	err = DeleteUsers(db, &models.User{Email: bond.Email})
	require.NoError(t, err)

	_, err = GetUser(db, &models.User{Email: bond.Email})
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent user should not fail
	err = DeleteUsers(db, &models.User{Email: bond.Email})
	require.NoError(t, err)

	// deleting an user should not delete unrelated users
	_, err = GetUser(db, &models.User{Email: bourne.Email})
	require.NoError(t, err)
}
