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

func ListGroups(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Group, error) {
	return list[models.Group](db, p, selectors...)
}

func ByGroupMember(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN identities_groups ON groups.id = identities_groups.group_id").
			Where("identities_groups.identity_id = ?", id)
	}
}

func DeleteGroups(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, &models.Pagination{}, selectors...)
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

		identities, err := ListIdentities(db, &models.Pagination{}, []SelectorFunc{ByOptionalIdentityGroupID(g.ID)}...)
		if err != nil {
			return err
		}

		var uidsToRemove []uid.ID
		for _, id := range identities {
			uidsToRemove = append(uidsToRemove, id.ID)
		}
		err = RemoveUsersFromGroup(db, g.ID, uidsToRemove)
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}

func AddUsersToGroup(db *gorm.DB, groupID uid.ID, idsToAdd []uid.ID) error {
	for _, id := range idsToAdd {
		err := db.Exec("INSERT INTO identities_groups (group_id, identity_id) select ?, ? WHERE NOT EXISTS (SELECT 1 FROM identities_groups WHERE group_id = ? AND identity_id = ?)", groupID, id, groupID, id).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveUsersFromGroup(db *gorm.DB, groupID uid.ID, idsToRemove []uid.ID) error {
	for _, id := range idsToRemove {
		err := db.Exec("DELETE FROM identities_groups WHERE identity_id = ? AND group_id = ?", id, groupID).Error
		if err != nil {
			return err
		}
	}
	return nil
}
