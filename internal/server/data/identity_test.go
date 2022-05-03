package data

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ssoroka/slice"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestIdentity(t *testing.T) {
	db := setup(t)

	bond := models.Identity{Name: "jbond@infrahq.com"}

	err := db.Create(&bond).Error
	assert.NilError(t, err)

	var identity models.Identity
	err = db.First(&identity, &models.Identity{Name: bond.Name}).Error
	assert.NilError(t, err)
	assert.Assert(t, 0 != identity.ID)
	assert.Equal(t, bond.Name, identity.Name)
}

func createIdentities(t *testing.T, db *gorm.DB, identities ...models.Identity) {
	for i := range identities {
		err := CreateIdentity(db, &identities[i])
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

	identity, err := GetIdentity(db, ByName(bond.Name))
	assert.NilError(t, err)
	assert.Assert(t, 0 != identity.ID)
}

func TestListIdentities(t *testing.T) {
	db := setup(t)
	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

	t.Run("list all", func(t *testing.T) {
		identities, err := ListIdentities(db)
		assert.NilError(t, err)
		expected := []models.Identity{bauer, bond, bourne}
		assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
	})

	t.Run("filter by name", func(t *testing.T) {
		identities, err := ListIdentities(db, ByName(bourne.Name))
		assert.NilError(t, err)
		expected := []models.Identity{bourne}
		assert.DeepEqual(t, identities, expected, cmpModelsIdentityShallow)
	})
}

var cmpModelsIdentityShallow = cmp.Comparer(func(x, y models.Identity) bool {
	return x.Name == y.Name
})

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

	// deleting a nonexistent identity should not fail
	err = DeleteIdentities(db, ByName(bond.Name))
	assert.NilError(t, err)

	// deleting a identity should not delete unrelated identities
	_, err = GetIdentity(db, ByName(bourne.Name))
	assert.NilError(t, err)
}

func TestReCreateIdentitySameName(t *testing.T) {
	db := setup(t)

	var (
		bond   = models.Identity{Name: "jbond@infrahq.com"}
		bourne = models.Identity{Name: "jbourne@infrahq.com"}
		bauer  = models.Identity{Name: "jbauer@infrahq.com"}
	)

	createIdentities(t, db, bond, bourne, bauer)

	err := DeleteIdentities(db, ByName(bond.Name))
	assert.NilError(t, err)

	err = CreateIdentity(db, &models.Identity{Name: bond.Name})
	assert.NilError(t, err)
}

func TestAssignIdentityToGroups(t *testing.T) {
	tests := []struct {
		Name           string
		StartingGroups []string // groups identity starts with
		ExistingGroups []string // groups from last provider sync
		IncomingGroups []string // groups from this provider sync
		ExpectedGroups []string // groups identity should have at end
	}{
		{
			Name:           "test where the provider is trying to add a group the identity doesn't have elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo2"},
			ExpectedGroups: []string{"foo", "foo2"},
		},
		{
			Name:           "test where the provider is trying to add a group the identity has from elsewhere",
			StartingGroups: []string{"foo"},
			ExistingGroups: []string{},
			IncomingGroups: []string{"foo", "foo2"},
			ExpectedGroups: []string{"foo", "foo2"},
		},
	}

	db := setup(t)

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// setup identity
			identity := &models.Identity{Name: fmt.Sprintf("foo+%d@example.com", i)}
			err := CreateIdentity(db, identity)
			assert.NilError(t, err)

			// setup identity's groups
			for _, gn := range test.StartingGroups {
				g, err := GetGroup(db, ByName(gn))
				if errors.Is(err, internal.ErrNotFound) {
					g = &models.Group{Name: gn}
					err = CreateGroup(db, g)
				}
				assert.NilError(t, err)
				identity.Groups = append(identity.Groups, *g)
			}
			err = SaveIdentity(db, identity)
			assert.NilError(t, err)

			// setup provuderUser record
			provider := InfraProvider(db)
			pu, err := CreateProviderUser(db, provider, identity)
			assert.NilError(t, err)

			pu.Groups = test.ExistingGroups
			err = UpdateProviderUser(db, pu)
			assert.NilError(t, err)

			err = AssignIdentityToGroups(db, identity, provider, test.IncomingGroups)
			assert.NilError(t, err)

			// reload identity and check groups
			id, err := GetIdentity(db.Preload("Groups"), ByID(identity.ID))
			assert.NilError(t, err)
			groupNames := slice.Map[models.Group, string](id.Groups, func(g models.Group) string {
				return g.Name
			})

			assert.DeepEqual(t, slice.Sort(groupNames), slice.Sort(test.ExpectedGroups))
		})
	}
}
