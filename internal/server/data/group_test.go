package data

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestGroup(t *testing.T) {
	db := setup(t)

	everyone := models.Group{Name: "Everyone"}

	err := db.Create(&everyone).Error
	assert.NilError(t, err)

	var group models.Group
	err = db.First(&group, &models.Group{Name: everyone.Name}).Error
	assert.NilError(t, err)
	assert.Assert(t, 0 != group.ID)
	assert.Equal(t, everyone.Name, group.Name)
}

func TestCreateGroup(t *testing.T) {
	db := setup(t)

	everyone := models.Group{Name: "Everyone"}

	err := CreateGroup(db, &everyone)
	assert.NilError(t, err)

	group := everyone
	assert.Assert(t, 0 != group.ID)
	assert.Equal(t, everyone.Name, group.Name)
}

func createGroups(t *testing.T, db *gorm.DB, groups ...models.Group) {
	for i := range groups {
		err := CreateGroup(db, &groups[i])
		assert.NilError(t, err)
	}
}

func TestCreateGroupDuplicate(t *testing.T) {
	var (
		everyone  = models.Group{Name: "Everyone"}
		engineers = models.Group{Name: "Engineering"}
		product   = models.Group{Name: "Product"}
	)

	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	err := CreateGroup(db, &models.Group{Name: "Everyone"})
	assert.ErrorContains(t, err, "duplicate record")
}

func TestGetGroup(t *testing.T) {
	db := setup(t)

	var (
		everyone  = models.Group{Name: "Everyone"}
		engineers = models.Group{Name: "Engineering"}
		product   = models.Group{Name: "Product"}
	)

	createGroups(t, db, everyone, engineers, product)

	group, err := GetGroup(db, ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, 0 != group.ID)
}

func TestListGroups(t *testing.T) {
	db := setup(t)

	var (
		everyone  = models.Group{Name: "Everyone"}
		engineers = models.Group{Name: "Engineering"}
		product   = models.Group{Name: "Product"}
	)

	createGroups(t, db, everyone, engineers, product)

	groups, err := ListGroups(db)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(groups))

	groups, err = ListGroups(db, ByName(engineers.Name))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestDeleteGroup(t *testing.T) {
	db := setup(t)

	var (
		everyone  = models.Group{Name: "Everyone"}
		engineers = models.Group{Name: "Engineering"}
		product   = models.Group{Name: "Product"}
	)

	createGroups(t, db, everyone, engineers, product)

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
}

func TestRecreateGroupSameName(t *testing.T) {
	db := setup(t)

	var (
		everyone  = models.Group{Name: "Everyone"}
		engineers = models.Group{Name: "Engineering"}
		product   = models.Group{Name: "Product"}
	)

	createGroups(t, db, everyone, engineers, product)

	err := DeleteGroups(db, ByName(everyone.Name))
	assert.NilError(t, err)

	err = CreateGroup(db, &models.Group{Name: everyone.Name})
	assert.NilError(t, err)
}
