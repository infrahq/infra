package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type groupsTable models.Group

func (g groupsTable) Table() string {
	return "groups"
}

func (g groupsTable) Columns() []string {
	return []string{"created_at", "created_by", "created_by_provider", "deleted_at", "id", "name", "organization_id", "updated_at"}
}

func (g groupsTable) Values() []any {
	return []any{g.CreatedAt, g.CreatedBy, g.CreatedByProvider, g.DeletedAt, g.ID, g.Name, g.OrganizationID, g.UpdatedAt}
}

func (g *groupsTable) ScanFields() []any {
	return []any{&g.CreatedAt, &g.CreatedBy, &g.CreatedByProvider, &g.DeletedAt, &g.ID, &g.Name, &g.OrganizationID, &g.UpdatedAt}
}

func CreateGroup(tx WriteTxn, group *models.Group) error {
	return insert(tx, (*groupsTable)(group))
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

func ListGroups(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Group, error) {
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

func groupIDsForUser(tx ReadTxn, userID uid.ID) ([]uid.ID, error) {
	stmt := `SELECT DISTINCT group_id FROM identities_groups WHERE identity_id = ?`
	rows, err := tx.Query(stmt, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []uid.ID
	for rows.Next() {
		var id uid.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func DeleteGroups(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, nil, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, g := range toDelete {
		ids = append(ids, g.ID)

		err := DeleteGrants(db, DeleteGrantsOptions{BySubject: g.PolyID()})
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

// TODO: do this with a join in ListGroups and GetGroup
func CountUsersInGroup(tx GormTxn, groupID uid.ID) (int64, error) {
	db := tx.GormDB()
	var count int64
	err := db.Table("identities_groups").Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
