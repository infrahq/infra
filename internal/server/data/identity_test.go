package data

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ssoroka/slice"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestUser(t *testing.T) {
	db := setup(t)

	bond := models.Identity{Name: "jbond@infrahq.com", Kind: models.UserKind}

	err := db.Create(&bond).Error
	assert.NilError(t, err)

	var user models.Identity
	err = db.First(&user, &models.Identity{Name: bond.Name, Kind: models.UserKind}).Error
	assert.NilError(t, err)
	assert.Assert(t, 0 != user.ID)
	assert.Equal(t, bond.Name, user.Name)
}

func createIdentities(t *testing.T, db *gorm.DB, users ...models.Identity) {
	for i := range users {
		err := CreateIdentity(db, &users[i])
		assert.NilError(t, err)
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	db := setup(t)

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

	b := bond
	b.ID = 0
	err := CreateIdentity(db, &b)
	assert.ErrorContains(t, err, "duplicate record")
}

func TestGetIdentity(t *testing.T) {
	db := setup(t)

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

	user, err := GetIdentity(db, ByName(bond.Name))
	assert.NilError(t, err)
	assert.Assert(t, 0 != user.ID)
}

func TestListIdentities(t *testing.T) {
	db := setup(t)

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

	users, err := ListIdentities(db)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(users))

	users, err = ListIdentities(db, ByName(bourne.Name))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(users))
}

func TestDeleteIdentity(t *testing.T) {
	db := setup(t)

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

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

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com", Kind: models.UserKind}
		bourne = models.Identity{Name: "jbourne@infrahq.com", Kind: models.UserKind}
		bauer  = models.Identity{Name: "jbauer@infrahq.com", Kind: models.UserKind}
	)

	createIdentities(t, db, bond, bourne, bauer)

	err := DeleteIdentities(db, ByName(bond.Name))
	assert.NilError(t, err)

	err = CreateIdentity(db, &models.Identity{Name: bond.Name, Kind: models.UserKind})
	assert.NilError(t, err)
}

func TestAssignIdentityToGroups(t *testing.T) {
	tests := []struct {
		Name           string
		StartingGroups []string // groups user starts with
		ExistingGroups []string // groups from last provider sync
		IncomingGroups []string // groups from this provider sync
		ExpectedGroups []string // groups user should have at end
	}{
		{
			Name:           "test where the provider is trying to add a group the user doesn't have elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo2"},
			ExpectedGroups: []string{"foo", "foo2"},
		},
		{
			Name:           "test where the provider is trying to add a group the user has from elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo", "foo2"},
			ExpectedGroups: []string{"foo", "foo2"},
		},
	}

	db := setup(t)

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// setup user
			user := &models.Identity{
				Name: fmt.Sprintf("foo+%d@example.com", i),
				Kind: models.UserKind,
			}
			err := CreateIdentity(db, user)
			assert.NilError(t, err)

			// setup user's groups
			for _, gn := range test.StartingGroups {
				g, err := GetGroup(db, ByName(gn))
				if errors.Is(err, internal.ErrNotFound) {
					g = &models.Group{Name: gn}
					err = CreateGroup(db, g)
				}
				assert.NilError(t, err)
				user.Groups = append(user.Groups, *g)
			}
			err = SaveIdentity(db, user)
			assert.NilError(t, err)

			// setup provuderUser record
			provider := InfraProvider(db)
			pu, err := CreateProviderUser(db, provider, user)
			assert.NilError(t, err)

			pu.Groups = test.ExistingGroups
			err = UpdateProviderUser(db, pu)
			assert.NilError(t, err)

			err = AssignIdentityToGroups(db, user, provider, test.IncomingGroups)
			assert.NilError(t, err)

			// reload user and check groups
			id, err := GetIdentity(db.Preload("Groups"), ByID(user.ID))
			assert.NilError(t, err)
			groupNames := slice.Map[models.Group, string](id.Groups, func(g models.Group) string {
				return g.Name
			})

			assert.DeepEqual(t, slice.Sort(groupNames), slice.Sort(test.ExpectedGroups))
		})
	}
}
