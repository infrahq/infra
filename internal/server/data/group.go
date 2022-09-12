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

type GetGroupOptions struct {
	// ByID instructs GetGroup to return the group matching this ID.
	ByID uid.ID
	// ByName instructs GetGroup to return the group matching this name.
	ByName string
}

func GetGroup(tx ReadTxn, opts GetGroupOptions) (*models.Group, error) {
	group := &groupsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(group))
	query.B("FROM groups")
	query.B("WHERE deleted_at is null")
	query.B("AND organization_id = ?", tx.OrganizationID())

	switch {
	case opts.ByID != 0:
		query.B("AND id = ?", opts.ByID)
	case opts.ByName != "":
		query.B("AND name = ?", opts.ByName)
	default:
		return nil, fmt.Errorf("GetGroup requires an ID")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(group.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}

	count, err := countUsersInGroup(tx, group.ID)
	if err != nil {
		return nil, err
	}
	group.TotalUsers = int(count)
	return (*models.Group)(group), nil
}

func ListGroups(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Group, error) {
	groups, err := list[models.Group](db, p, selectors...)
	if err != nil {
		return nil, err
	}

	for i := range groups {
		count, err := countUsersInGroup(db, groups[i].ID)
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

func AddUsersToGroup(tx WriteTxn, groupID uid.ID, idsToAdd []uid.ID) error {
	query := querybuilder.New("INSERT INTO identities_groups(group_id, identity_id)")
	query.B("VALUES")
	for i, id := range idsToAdd {
		query.B("(?, ?)", groupID, id)
		if i+1 != len(idsToAdd) {
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

func countUsersInGroup(tx ReadTxn, groupID uid.ID) (int64, error) {
	stmt := `SELECT count(*) FROM identities_groups WHERE group_id = ?`
	var count int64
	err := tx.QueryRow(stmt, groupID).Scan(&count)
	return count, handleError(err)
}
