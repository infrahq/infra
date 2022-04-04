package data

import (
	"sort"
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestGroup(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	everyone := models.Group{Name: "Everyone", ProviderID: providerID}

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

	providerID := uid.New()

	everyone := models.Group{Name: "Everyone", ProviderID: providerID}

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
	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	db := setup(t)
	createGroups(t, db, everyone, engineers, product)

	err := CreateGroup(db, &models.Group{Name: "Everyone", ProviderID: providerID})
	assert.Assert(t, is.Contains(err.Error(), "duplicate record"))
}

func TestGetGroup(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	group, err := GetGroup(db, ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, 0 != group.ID)
}

func TestListGroups(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	groups, err := ListGroups(db)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(groups))

	groups, err = ListGroups(db, ByName(engineers.Name))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestBindGroupIdentities(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bond      = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateIdentity(db, &bond)
	assert.NilError(t, err)

	groups, err := ListGroups(db)
	assert.NilError(t, err)

	for i := range groups {
		err := BindGroupIdentities(db, &groups[i], bond)
		assert.NilError(t, err)
	}

	user, err := GetIdentity(db.Preload("Groups"), ByName(bond.Name))
	assert.NilError(t, err)
	expected := []string{engineers.Name, everyone.Name, product.Name}
	actual := []string{
		user.Groups[0].Name,
		user.Groups[1].Name,
		user.Groups[2].Name,
	}
	sort.Strings(actual)
	assert.DeepEqual(t, actual, expected)
}

func TestGroupBindMoreIdentities(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bond      = models.Identity{Name: "jbond@infrahq.com", ProviderID: providerID}
		bourne    = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateIdentity(db, &bond)
	assert.NilError(t, err)

	group, err := GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 0))

	err = BindGroupIdentities(db, group, bond)
	assert.NilError(t, err)

	group, err = GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 1))

	err = CreateIdentity(db, &bourne)
	assert.NilError(t, err)

	err = BindGroupIdentities(db, group, bond, bourne)
	assert.NilError(t, err)

	group, err = GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 2))
}

func TestGroupBindLessIdentities(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
		bourne    = models.Identity{Name: "jbourne@infrahq.com", ProviderID: providerID}
		bauer     = models.Identity{Name: "jbauer@infrahq.com", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := CreateIdentity(db, &bourne)
	assert.NilError(t, err)

	err = CreateIdentity(db, &bauer)
	assert.NilError(t, err)

	group, err := GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 0))

	err = BindGroupIdentities(db, group, bourne, bauer)
	assert.NilError(t, err)

	group, err = GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 2))

	err = BindGroupIdentities(db, group, bauer)
	assert.NilError(t, err)

	group, err = GetGroup(db.Preload("Identities"), ByName(everyone.Name))
	assert.NilError(t, err)
	assert.Assert(t, is.Len(group.Identities, 1))
}

func TestDeleteGroup(t *testing.T) {
	db := setup(t)

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
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

	providerID := uid.New()

	var (
		everyone  = models.Group{Name: "Everyone", ProviderID: providerID}
		engineers = models.Group{Name: "Engineering", ProviderID: providerID}
		product   = models.Group{Name: "Product", ProviderID: providerID}
	)

	createGroups(t, db, everyone, engineers, product)

	err := DeleteGroups(db, ByName(everyone.Name))
	assert.NilError(t, err)

	err = CreateGroup(db, &models.Group{Name: everyone.Name})
	assert.NilError(t, err)
}
