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
	db = db.Order("name ASC")
	return list[models.Group](db, selectors...)
}

func ByGroupMember(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN identities_groups ON groups.id = identities_groups.group_id").
			Where("identities_groups.identity_id = ?", id)
	}
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

		identities, err := ListIdentitiesByGroup(db, g.ID, []SelectorFunc{}...)
		if err != nil {
			return err
		}

		err = RemoveUsersFromGroup(db, g.ID, identities)
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}

func AddUsersToGroup(db *gorm.DB, groupID uid.ID, identities []models.Identity) error {
	for _, id := range identities {
		err := db.Exec("insert into identities_groups (identity_id, group_id) values (?, ?)", id.ID, groupID).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveUsersFromGroup(db *gorm.DB, groupID uid.ID, identities []models.Identity) error {
	for _, id := range identities {
		err := db.Exec("delete from identities_groups where identity_id = ? and group_id = ?", id.ID, groupID).Error
		if err != nil {
			return err
		}
	}
	return nil
}
