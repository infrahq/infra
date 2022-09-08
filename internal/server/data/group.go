package data

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
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
	stmt := `
		SELECT DISTINCT group_id FROM identities_groups
		WHERE identity_id = ?
	`
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

func DeleteGroup(tx WriteTxn, id uid.ID) error {
	err := DeleteGrants(tx, DeleteGrantsOptions{BySubject: uid.NewGroupPolymorphicID(id)})
	if err != nil {
		return fmt.Errorf("remove grants: %w", err)
	}

	_, err = tx.Exec(`DELETE from identities_groups WHERE group_id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove users from group: %w", err)
	}

	stmt := `
		UPDATE groups
		SET deleted_at = ?
		WHERE id = ?
		AND deleted_at is null
		AND organization_id = ?`
	_, err = tx.Exec(stmt, time.Now(), id, tx.OrganizationID())
	return handleError(err)
}

func AddUsersToGroup(tx WriteTxn, groupID uid.ID, providerGroupName string, providerID uid.ID, idsToAdd []uid.ID) error {
	query := querybuilder.New("INSERT INTO identities_groups(group_id, identity_id, provider_id, provider_group_name)")
	query.B("VALUES")
	for i, id := range idsToAdd {
		query.B("(?, ?, ?, ?)", groupID, id, providerID, providerGroupName)
		if i+1 != len(idsToAdd) {
			query.B(",")
		}
	}
	query.B("ON CONFLICT DO NOTHING")

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

func AddUserToGroups(tx WriteTxn, providerID uid.ID, identityID uid.ID, groups []models.Group) error {
	query := querybuilder.New("INSERT INTO identities_groups(provider_id, identity_id, group_id, provider_group_name)")
	query.B("VALUES")
	for i, group := range groups {
		query.B("(?, ?, ?, ?)", providerID, identityID, group.ID, group.Name)
		if i+1 != len(groups) {
			query.B(",")
		}
	}
	query.B("ON CONFLICT DO NOTHING")

	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

// RemoveUsersFromGroup removes any user ID listed in idsToRemove from the group
// with ID groupID.
// Note that DeleteGroup also removes users from the group.
func RemoveUsersFromGroup(tx WriteTxn, groupID uid.ID, idsToRemove []uid.ID) error {
	stmt := `DELETE FROM identities_groups WHERE group_id = ? AND identity_id IN (?)`
	_, err := tx.Exec(stmt, groupID, idsToRemove)
	return handleError(err)
}

func CountUsersInGroup(tx GormTxn, groupID uid.ID) (int64, error) {
	db := tx.GormDB()
	var count int64
	err := db.
		Table("identities_groups").
		Where("group_id = ?", groupID).
		Distinct("identity_id").
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
