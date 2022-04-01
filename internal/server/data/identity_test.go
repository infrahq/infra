package data

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestUser(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	bond := models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID, Kind: models.UserKind}

	err := db.Create(&bond).Error
	require.NoError(t, err)

	var user models.Identity
	err = db.First(&user, &models.Identity{Name: bond.Name, Kind: models.UserKind}).Error
	require.NoError(t, err)
	require.NotEqual(t, 0, user.ID)
	require.Equal(t, bond.Name, user.Name)
}

func CreateIdentitys(t *testing.T, db *gorm.DB, users ...models.Identity) {
	for i := range users {
		err := CreateIdentity(db, &users[i])
		require.NoError(t, err)
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
		bourne = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID}
	)

	CreateIdentitys(t, db, bond, bourne, bauer)

	b := bond
	b.ID = 0
	err := CreateIdentity(db, &b)
	require.Contains(t, err.Error(), "duplicate record")
}

func TestGetIdentity(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
		bourne = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID}
	)

	CreateIdentitys(t, db, bond, bourne, bauer)

	user, err := GetIdentity(db, ByName(bond.Name))
	require.NoError(t, err)
	require.NotEqual(t, 0, user.ID)
}

func TestListIdentities(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
		bourne = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID}
	)

	CreateIdentitys(t, db, bond, bourne, bauer)

	users, err := ListIdentities(db)
	require.NoError(t, err)
	require.Equal(t, 3, len(users))

	users, err = ListIdentities(db, ByName(bourne.Name))
	require.NoError(t, err)
	require.Equal(t, 1, len(users))
}

func TestDeleteIdentity(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
		bourne = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID}
	)

	CreateIdentitys(t, db, bond, bourne, bauer)

	_, err := GetIdentity(db, ByName(bond.Name))
	require.NoError(t, err)

	err = DeleteIdentities(db, ByName(bond.Name))
	require.NoError(t, err)

	_, err = GetIdentity(db, ByName(bond.Name))
	require.EqualError(t, err, "record not found")

	// deleting a nonexistent user should not fail
	err = DeleteIdentities(db, ByName(bond.Name))
	require.NoError(t, err)

	// deleting a user should not delete unrelated users
	_, err = GetIdentity(db, ByName(bourne.Name))
	require.NoError(t, err)
}

func TestReCreateIdentitySameEmail(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID, Kind: models.UserKind}
		bourne = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID, Kind: models.UserKind}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID, Kind: models.UserKind}
	)

	CreateIdentitys(t, db, bond, bourne, bauer)

	err := DeleteIdentities(db, ByName(bond.Name))
	require.NoError(t, err)

	err = CreateIdentity(db, &models.Identity{Name: bond.Name, Kind: models.UserKind})
	require.NoError(t, err)
}
