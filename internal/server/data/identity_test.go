package data

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestUser(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	bond := models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID, Kind: models.UserKind}

	err := db.Create(&bond).Error
	assert.NilError(t, err)

	var user models.Identity
	err = db.First(&user, &models.Identity{Name: bond.Name, Kind: models.UserKind}).Error
	assert.NilError(t, err)
	assert.Assert(t, 0 != user.ID)
	assert.Equal(t, bond.Name, user.Name)
}

func CreateIdentitys(t *testing.T, db *gorm.DB, users ...models.Identity) {
	for i := range users {
		err := CreateIdentity(db, &users[i])
		assert.NilError(t, err)
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
	assert.Assert(t, is.Contains(err.Error(), "duplicate record"))
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
	assert.NilError(t, err)
	assert.Assert(t, 0 != user.ID)
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
	assert.NilError(t, err)
	assert.Equal(t, 3, len(users))

	users, err = ListIdentities(db, ByName(bourne.Name))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(users))
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
	assert.NilError(t, err)

	err = DeleteIdentities(db, ByName(bond.Name))
	assert.NilError(t, err)

	_, err = GetIdentity(db, ByName(bond.Name))
	assert.Error(t, err, "record not found")

	// deleting a nonexistent user should not fail
	err = DeleteIdentities(db, ByName(bond.Name))
	assert.NilError(t, err)

	// deleting a user should not delete unrelated users
	_, err = GetIdentity(db, ByName(bourne.Name))
	assert.NilError(t, err)
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
	assert.NilError(t, err)

	err = CreateIdentity(db, &models.Identity{Name: bond.Name, Kind: models.UserKind})
	assert.NilError(t, err)
}
