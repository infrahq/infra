package data

import (
	"testing"

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
	require.NotEqual(t, 0, user.ID)
	require.Equal(t, bond.Email, user.Email)
}

func TestCreateUser(t *testing.T) {
	db := setup(t)

	err := CreateUser(db, &bond)
	require.NoError(t, err)
	require.NotEqual(t, 0, bond.ID)
}

func createUsers(t *testing.T, db *gorm.DB, users ...models.User) {
	for i := range users {
		err := CreateUser(db, &users[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	b := bond
	b.ID = 0
	err := CreateUser(db, &b)
	require.Contains(t, err.Error(), "duplicate record")
}

func TestCreateOrUpdateUserCreate(t *testing.T) {
	db := setup(t)

	user, err := CreateOrUpdateUser(db, &bond)
	require.NoError(t, err)
	require.NotEqual(t, 0, user.ID)
	require.Equal(t, bond.Email, user.Email)
}

func TestCreateOrUpdateUserUpdate(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	bond.ID = 0
	bond.Email = "james@infrahq.com"
	user, err := CreateOrUpdateUser(db, &bond)
	require.NoError(t, err)
	require.NotEqual(t, 0, user.ID)
	require.Equal(t, "james@infrahq.com", user.Email)
}

func TestGetUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	user, err := GetUser(db, ByEmail(bond.Email))
	require.NoError(t, err)
	require.NotEqual(t, 0, user.ID)
}

func TestListUsers(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	users, err := ListUsers(db)
	require.NoError(t, err)
	require.Equal(t, 3, len(users))

	users, err = ListUsers(db, ByEmail(bourne.Email))
	require.NoError(t, err)
	require.Equal(t, 1, len(users))
}

func TestDeleteUser(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	_, err := GetUser(db, ByEmail(bond.Email))
	require.NoError(t, err)

	err = DeleteUsers(db, ByEmail(bond.Email))
	require.NoError(t, err)

	_, err = GetUser(db, ByEmail(bond.Email))
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent user should not fail
	err = DeleteUsers(db, ByEmail(bond.Email))
	require.NoError(t, err)

	// deleting an user should not delete unrelated users
	_, err = GetUser(db, ByEmail(bourne.Email))
	require.NoError(t, err)
}

func TestRecreateUserSameEmail(t *testing.T) {
	db := setup(t)
	createUsers(t, db, bond, bourne, bauer)

	err := DeleteUsers(db, ByEmail(bond.Email))
	require.NoError(t, err)

	err = CreateUser(db, &models.User{Email: bond.Email})
	require.NoError(t, err)
}
