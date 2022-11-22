package data

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type organizationsTable models.Organization

func (organizationsTable) Table() string {
	return "organizations"
}

func (o organizationsTable) Columns() []string {
	return []string{"created_at", "created_by", "deleted_at", "domain", "id", "name", "updated_at", "allowed_domains"}
}

func (o organizationsTable) Values() []any {
	return []any{o.CreatedAt, o.CreatedBy, o.DeletedAt, o.Domain, o.ID, o.Name, o.UpdatedAt, o.AllowedDomains}
}

func (o *organizationsTable) ScanFields() []any {
	return []any{&o.CreatedAt, &o.CreatedBy, &o.DeletedAt, &o.Domain, &o.ID, &o.Name, &o.UpdatedAt, &o.AllowedDomains}
}

// CreateOrganization creates a new organization, and initializes it with
// settings, an infra provider, a connector user, and a grant for the connector.
func CreateOrganization(tx WriteTxn, org *models.Organization) error {
	if org.Name == "" {
		return fmt.Errorf("Organization.Name is required")
	}
	if err := insert(tx, (*organizationsTable)(org)); err != nil {
		return fmt.Errorf("creating org: %w", err)
	}
	if err := createSettings(tx, org.ID); err != nil {
		return fmt.Errorf("initializing org settings: %w", err)
	}

	infraProvider := &models.Provider{
		Name:               models.InternalInfraProviderName,
		Kind:               models.ProviderKindInfra,
		CreatedBy:          models.CreatedBySystem,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	if err := CreateProvider(tx, infraProvider); err != nil {
		return fmt.Errorf("failed to create infra provider: %w", err)
	}

	connector := &models.Identity{
		Name:               models.InternalInfraConnectorIdentityName,
		CreatedBy:          models.CreatedBySystem,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	// this identity is used to create access keys for connectors
	if err := CreateIdentity(tx, connector); err != nil {
		return fmt.Errorf("failed to create connector identity while creating org: %w", err)
	}

	err := CreateGrant(tx, &models.Grant{
		Subject:            uid.NewIdentityPolymorphicID(connector.ID),
		Privilege:          models.InfraConnectorRole,
		Resource:           "infra",
		CreatedBy:          models.CreatedBySystem,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	})
	if err != nil {
		return fmt.Errorf("failed to grant connector role creating org: %w", err)
	}

	return nil
}

type GetOrganizationOptions struct {
	ByID     uid.ID
	ByDomain string
}

func GetOrganization(tx ReadTxn, opts GetOrganizationOptions) (*models.Organization, error) {
	org := organizationsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(org))
	query.B("FROM organizations")
	query.B("WHERE deleted_at is NULL")

	switch {
	case opts.ByID != 0:
		query.B("AND id = ?", opts.ByID)
	case opts.ByDomain != "":
		query.B("AND domain = ?", opts.ByDomain)
	default:
		return nil, fmt.Errorf("an ID or domain is required to get organization")
	}

	err := tx.QueryRow(query.String(), query.Args...).Scan(org.ScanFields()...)
	if err != nil {
		return nil, handleError(err)
	}
	return (*models.Organization)(&org), nil
}

type ListOrganizationsOptions struct {
	ByName string

	Pagination *Pagination
}

func ListOrganizations(tx ReadTxn, opts ListOrganizationsOptions) ([]models.Organization, error) {
	table := organizationsTable{}
	query := querybuilder.New("SELECT")
	query.B(columnsForSelect(table))
	if opts.Pagination != nil {
		query.B(", count(*) OVER()")
	}
	query.B("FROM organizations")
	query.B("WHERE deleted_at is NULL")

	if opts.ByName != "" {
		query.B("AND name = ?", opts.ByName)
	}
	query.B("ORDER BY id ASC")
	if opts.Pagination != nil {
		opts.Pagination.PaginateQuery(query)
	}

	rows, err := tx.Query(query.String(), query.Args...)
	if err != nil {
		return nil, err
	}
	return scanRows(rows, func(org *models.Organization) []any {
		fields := (*organizationsTable)(org).ScanFields()
		if opts.Pagination != nil {
			fields = append(fields, &opts.Pagination.TotalCount)
		}
		return fields
	})
}

func DeleteOrganization(tx WriteTxn, id uid.ID) error {
	// TODO: delete everything in the organization

	stmt := `
		UPDATE organizations
		SET deleted_at = ?
		WHERE id = ? AND deleted_at is NULL`

	_, err := tx.Exec(stmt, time.Now(), id)
	return err
}

func UpdateOrganization(tx WriteTxn, org *models.Organization) error {
	return update(tx, (*organizationsTable)(org))
}

func CountOrganizations(tx ReadTxn) (int64, error) {
	return countRows(tx, organizationsTable{})
}
