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

func ByGroupMember(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN identities_groups ON groups.id = identities_groups.group_id").
			Where("identities_groups.identity_id = ?", id)
	}
}

func DeleteGroups(db *gorm.DB, orgID uid.ID, selectors ...SelectorFunc) error {
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

		identities, err := ListIdentities(db, []SelectorFunc{ByOptionalIdentityGroupID(g.ID)}...)
		if err != nil {
			return err
		}

		var uidsToRemove []uid.ID
		for _, id := range identities {
			uidsToRemove = append(uidsToRemove, id.ID)
		}
		err = RemoveUsersFromGroup(db, orgID, g.ID, uidsToRemove)
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}

func AddUsersToGroup(db *gorm.DB, orgID, groupID uid.ID, idsToAdd []uid.ID) error {
	for _, id := range idsToAdd {
		// This is effectively an "INSERT OR IGNORE" or "INSERT ... ON CONFLICT ... DO NOTHING" statement which
		// works across both sqlite and postgres
		err := db.Exec("INSERT INTO identities_groups (organization_id, group_id, identity_id) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM identities_groups WHERE organization_id = ? AND group_id = ? AND identity_id = ?)", orgID, groupID, id, orgID, groupID, id).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveUsersFromGroup(db *gorm.DB, orgID, groupID uid.ID, idsToRemove []uid.ID) error {
	for _, id := range idsToRemove {
		err := db.Exec("DELETE FROM identities_groups WHERE organization_id = ? AND identity_id = ? AND group_id = ?", orgID, id, groupID).Error
		if err != nil {
			return err
		}
	}
	return nil
}
