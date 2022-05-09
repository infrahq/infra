package data

import (
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		everyone := models.Group{Name: "Everyone"}

		err := CreateGroup(db, &everyone)
		assert.NilError(t, err)

		group := everyone
		assert.Assert(t, 0 != group.ID)
		assert.Equal(t, everyone.Name, group.Name)
	})
}

func createGroups(t *testing.T, db *gorm.DB, groups ...*models.Group) {
	t.Helper()
	for i := range groups {
		err := CreateGroup(db, groups[i])
		assert.NilError(t, err, groups[i].Name)
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		var (
			everyone  = models.Group{Name: "Everyone"}
			engineers = models.Group{Name: "Engineering"}
			product   = models.Group{Name: "Product"}
		)

		createGroups(t, db, &everyone, &engineers, &product)

		err := CreateGroup(db, &models.Group{Name: "Everyone"})
		assert.ErrorContains(t, err, "duplicate record")
	})
}

func TestGetGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
			actual, err := ListGroups(db)
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
				{Name: "Everyone"},
				{Name: "Product"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by name", func(t *testing.T) {
			actual, err := ListGroups(db, ByName(engineers.Name))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Engineering"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})

		t.Run("filter by identity membership", func(t *testing.T) {
			actual, err := ListGroups(db, WhereGroupIncludesUser(firstUser.ID))
			assert.NilError(t, err)
			expected := []models.Group{
				{Name: "Everyone"},
				{Name: "Engineering"},
			}
			assert.DeepEqual(t, actual, expected, cmpGroupShallow)
		})
	})
}

var cmpGroupShallow = gocmp.Comparer(func(x, y models.Group) bool {
	return x.Name == y.Name
})

func TestDeleteGroup(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
