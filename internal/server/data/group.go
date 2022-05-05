package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGroup(db *gorm.DB, group *models.Group) error {
	return add(db, group)
}

func GetGroup(db *gorm.DB, selectors ...SelectorFunc) (*models.Group, error) {
	return get[models.Group](db, selectors...)
}

func ListGroups(db *gorm.DB, selectors ...SelectorFunc) ([]models.Group, error) {
	return list[models.Group](db, selectors...)
}

func WhereGroupIncludesUser(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN identities_groups ON groups.id = identities_groups.group_id").
			Where("identities_groups.identity_id = ?", id)
	}
}

// TODO: replace calls with ListGroups(..., WhereGroupIncludesUser(id))
func ListIdentityGroups(db *gorm.DB, identityID uid.ID) ([]models.Group, error) {
	var result []models.Group
	err := db.Model(&models.Identity{Model: models.Model{ID: identityID}}).Association("Groups").Find(&result)
	return result, err
}

func DeleteGroups(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, g := range toDelete {
		ids = append(ids, g.ID)

		err := DeleteGrants(db, BySubject(g.PolyID()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}
