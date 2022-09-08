package data

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type providerGroupTable models.ProviderGroup

func (p providerGroupTable) Table() string {
	return "provider_groups"
}

func (p providerGroupTable) Columns() []string {
	return []string{"organization_id", "created_at", "updated_at", "provider_id", "name"}
}

func (p providerGroupTable) Values() []any {
	return []any{p.OrganizationID, p.CreatedAt, p.UpdatedAt, p.ProviderID, p.Name}
}

func (p *providerGroupTable) ScanFields() []any {
	return []any{&p.OrganizationID, &p.CreatedAt, &p.UpdatedAt, &p.ProviderID, &p.Name}
}

func (pg *providerGroupTable) OnInsert() error {
	if pg.CreatedAt.IsZero() {
		pg.CreatedAt = time.Now()
	}
	pg.UpdatedAt = pg.CreatedAt
	return nil
}

func (pg *providerGroupTable) OnUpdate() error {
	pg.UpdatedAt = time.Now()
	return nil
}

// CreateProviderGroup adds a database entity for tracking group members at a provider
func CreateProviderGroup(db WriteTxn, providerGroup *models.ProviderGroup) error {
	switch {
	case providerGroup.ProviderID == 0:
		return fmt.Errorf("providerID is required")
	case providerGroup.Name == "":
		return fmt.Errorf("name is required")
	}

	providerGroup.OrganizationID = db.OrganizationID()

	return insert(db, (*providerGroupTable)(providerGroup))
}

// GetProviderGroup returns the group with the specified name for a provider
func GetProviderGroup(tx ReadTxn, providerID uid.ID, name string) (*models.ProviderGroup, error) {
	providerGroup := &providerGroupTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(providerGroup))
	query.B("FROM")
	query.B(providerGroup.Table())
	query.B("WHERE organization_id = ?", tx.OrganizationID())
	query.B("AND provider_id = ?", providerID)
	query.B("AND name = ?", name)

	err := tx.QueryRow(query.String(), query.Args...).Scan(providerGroup.ScanFields()...)
	if err != nil {
		return nil, handleReadError(err)
	}
	pg := (*models.ProviderGroup)(providerGroup)

	return pg, nil
}

type ListProviderGroupOptions struct {
	ByProviderID       uid.ID
	ByMemberIdentityID uid.ID
}

// ListProviderGroups returns all provider groups that match the specified criteria
func ListProviderGroups(tx ReadTxn, opts ListProviderGroupOptions) ([]models.ProviderGroup, error) {
	table := &providerGroupTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	query.B("FROM")
	query.B(table.Table())

	if opts.ByMemberIdentityID != 0 {
		query.B(`
			JOIN provider_groups_provider_users 
			ON provider_groups.name = provider_groups_provider_users.provider_group_name 
			AND provider_groups.provider_id = provider_groups_provider_users.provider_id
			`)
	}

	query.B("WHERE organization_id = ?", tx.OrganizationID())
	if opts.ByProviderID != 0 {
		query.B("AND provider_groups.provider_id = ?", opts.ByProviderID)
	}
	if opts.ByMemberIdentityID != 0 {
		query.B(`AND provider_groups_provider_users.provider_user_identity_id = ?`, opts.ByMemberIdentityID)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ProviderGroup
	for rows.Next() {
		var providerGroup models.ProviderGroup
		fields := (*providerGroupTable)(&providerGroup).ScanFields()

		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}

		result = append(result, providerGroup)
	}

	return result, rows.Err()
}

// addMemberToProviderGroups adds a link between a provider user and group that exists in a provider
func addMemberToProviderGroups(tx GormTxn, user *models.ProviderUser, providerGroupNames []string) error {
	// the org does not need to be set here since all provider users and groups are org specific
	query := querybuilder.New("INSERT INTO provider_groups_provider_users (provider_id, provider_user_identity_id, provider_group_name)")
	query.B("VALUES")

	for i, grpName := range providerGroupNames {
		query.B("(?, ?, ?)", user.ProviderID, user.IdentityID, grpName)
		if i+1 != len(providerGroupNames) {
			query.B(",")
		}
	}
	query.B("ON CONFLICT DO NOTHING")

	_, err := tx.Exec(query.String(), query.Args...)
	if err != nil {
		return fmt.Errorf("add member to provider groups %w", err)
	}

	return nil
}

func removeMemberFromProviderGroups(tx GormTxn, user *models.ProviderUser, providerGroupNames []string) error {
	// the org does not need to be set here since all provider users and groups are org specific
	_, err := tx.Exec(`
	DELETE FROM provider_groups_provider_users
	WHERE provider_user_identity_id = ? AND provider_id = ? AND provider_group_name IN ?
	`,
		user.IdentityID, user.ProviderID, providerGroupNames)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM identities_groups
	WHERE identity_id = ? AND provider_id = ? AND provider_group_name IN ?
	`,
		user.IdentityID, user.ProviderID, providerGroupNames)

	return err
}
