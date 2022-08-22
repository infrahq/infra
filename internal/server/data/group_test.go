package data

import (
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		everyone := models.Group{Name: "Everyone"}

		err := db.Create(&everyone).Error
		assert.NilError(t, err)

		var group models.Group
		err = db.First(&group, &models.Group{Name: everyone.Name}).Error
		assert.NilError(t, err)
		assert.Assert(t, 0 != group.ID)
		assert.Equal(t, everyone.Name, group.Name)
	})
}

func TestCreateGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		everyone := models.Group{Name: "Everyone"}

		err := CreateGroup(db, &everyone)
		assert.NilError(t, err)

		group := everyone
		assert.Assert(t, 0 != group.ID)
		assert.Equal(t, everyone.Name, group.Name)
	})
}

func createGroups(t *testing.T, db GormTxn, groups ...*models.Group) {
	t.Helper()
	for i := range groups {
		err := CreateGroup(db, groups[i])
		assert.NilError(t, err, groups[i].Name)
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		err := CreateGroup(db, &models.Group{Name: "Everyone"})
		assert.ErrorContains(t, err, "a group with that name already exists")
	})
}

func TestGetGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		group, err := GetGroup(db, ByName(everyone.Name))
		assert.NilError(t, err)
		assert.Assert(t, 0 != group.ID)
	})
}

func TestListGroups(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		firstUser := models.Identity{
			Name:   "firstly",
			Groups: []models.Group{everyone, engineers},
		}
		secondUser := models.Identity{
			Name:   "secondarly",
			Groups: []models.Group{everyone, product},
		}
		createIdentities(t, db, &firstUser, &secondUser)

		t.Run("all", func(t *testing.T) {
			actual, err := ListGroups(db, nil)
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
				{Name: "Everyone"},
				{Name: "Product"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by name", func(t *testing.T) {
			actual, err := ListGroups(db, nil, ByName(engineers.Name))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by identity membership", func(t *testing.T) {
			actual, err := ListGroups(db, nil, ByGroupMember(firstUser.ID))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
				{Name: "Everyone"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})
	})
}

var cmpGroupShallow = gocmp.Comparer(func(x, y models.Group) bool {
	return x.Name == y.Name
})

func TestDeleteGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		_, err := GetGroup(db, ByName(everyone.Name))
		assert.NilError(t, err)

		err = DeleteGroups(db, ByName(everyone.Name))
		assert.NilError(t, err)

		_, err = GetGroup(db, ByName(everyone.Name))
		assert.Error(t, err, "record not found")

		// deleting a nonexistent group should not fail
		err = DeleteGroups(db, ByName(everyone.Name))
		assert.NilError(t, err)

		// deleting an group should not delete unrelated groups
		_, err = GetGroup(db, ByName(engineers.Name))
		assert.NilError(t, err)
	})
}

func TestRecreateGroupSameName(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		err := DeleteGroups(db, ByName(everyone.Name))
		assert.NilError(t, err)

		err = CreateGroup(db, &models.Group{Name: everyone.Name})
		assert.NilError(t, err)
	})
}

func TestAddUsersToGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		var everyone = models.Group{Name: "Everyone"}
		createGroups(t, db, &everyone)

		var (
			bond = models.Identity{
				Name:   "jbond@infrahq.com",
				Groups: []models.Group{everyone},
			}
			bourne = models.Identity{
				Name:   "jbourne@infrahq.com",
				Groups: []models.Group{},
			}
			bauer = models.Identity{Name: "jbauer@infrahq.com",
				Groups: []models.Group{},
			}
		)

		createIdentities(t, db, &bond, &bourne, &bauer)

		t.Run("add identities to group", func(t *testing.T) {
			actual, err := ListIdentities(db, nil, []SelectorFunc{ByOptionalIdentityGroupID(everyone.ID)}...)
			assert.NilError(t, err)
			expected := []models.Identity{bond}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)

			err = AddUsersToGroup(db, everyone.ID, []uid.ID{bourne.ID, bauer.ID})
			assert.NilError(t, err)

			actual, err = ListIdentities(db, nil, []SelectorFunc{ByOptionalIdentityGroupID(everyone.ID)}...)
			assert.NilError(t, err)
			expected = []models.Identity{bauer, bond, bourne}
			assert.DeepEqual(t, actual, expected, cmpModelsIdentityShallow)
		})
	})
}
