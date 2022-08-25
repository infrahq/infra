package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGroup(db GormTxn, group *models.Group) error {
	return add(db, group)
}

func GetGroup(db GormTxn, selectors ...SelectorFunc) (*models.Group, error) {
	group, err := get[models.Group](db, selectors...)
	if err != nil {
		return nil, err
	}

	count, err := CountUsersInGroup(db, group.ID)
	if err != nil {
		return nil, err
	}
	group.TotalUsers = int(count)
	return group, nil
}

func ListGroups(db GormTxn, p *models.Pagination, selectors ...SelectorFunc) ([]models.Group, error) {
	groups, err := list[models.Group](db, p, selectors...)

	if err != nil {
		return nil, err
	}

	for i := range groups {
		count, err := CountUsersInGroup(db, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].TotalUsers = int(count)
	}

	return groups, nil

}

func ByGroupMember(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN identities_groups ON groups.id = identities_groups.group_id").
			Where("identities_groups.identity_id = ?", id)
	}
}

func DeleteGroups(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, nil, selectors...)
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

		identities, err := ListIdentities(db, nil, []SelectorFunc{ByOptionalIdentityGroupID(g.ID)}...)
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

func AddUsersToGroup(db GormTxn, groupID uid.ID, idsToAdd []uid.ID) error {
	for _, id := range idsToAdd {
		// This is effectively an "INSERT OR IGNORE" or "INSERT ... ON CONFLICT ... DO NOTHING" statement which
		// works across both sqlite and postgres
		_, err := db.Exec("INSERT INTO identities_groups (group_id, identity_id) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM identities_groups WHERE group_id = ? AND identity_id = ?)", groupID, id, groupID, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveUsersFromGroup(db GormTxn, groupID uid.ID, idsToRemove []uid.ID) error {
	for _, id := range idsToRemove {
		_, err := db.Exec("DELETE FROM identities_groups WHERE identity_id = ? AND group_id = ?", id, groupID)
		if err != nil {
			return err
		}
	}
	return nil
}

func CountUsersInGroup(tx GormTxn, groupID uid.ID) (int64, error) {
	db := tx.GormDB()
	var count int64
	err := db.Table("identities_groups").Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
