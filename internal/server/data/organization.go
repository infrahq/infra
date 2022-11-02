package data

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type organizationsTable models.Organization

func (organizationsTable) Table() string {
	return "organizations"
}

func (o organizationsTable) Columns() []string {
	return []string{"created_at", "created_by", "deleted_at", "domain", "id", "name", "updated_at"}
}

func (o organizationsTable) Values() []any {
	return []any{o.CreatedAt, o.CreatedBy, o.DeletedAt, o.Domain, o.ID,

		o.Name, o.UpdatedAt}
}

func (o *organizationsTable) ScanFields() []any {
	return []any{&o.CreatedAt, &o.CreatedBy, &o.DeletedAt, &o.Domain, &o.ID, &o.Name, &o.UpdatedAt}
}

// CreateOrganization creates a new organization, and initializes it with
// settings, an infra provider, a connector user, and a grant for the connector.
func CreateOrganization(tx GormTxn, org *models.Organization) error {
	if org.Name == "" {
		return fmt.Errorf("Organization.Name is required")
	}
	err := insert(tx, (*organizationsTable)(org))
	if err != nil {
		return fmt.Errorf("creating org: %w", err)
	}

	_, err = initializeSettings(tx, org.ID)
	if err != nil {
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

	err = CreateGrant(tx, &models.Grant{
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

func GetOrganization(db GormTxn, selectors ...SelectorFunc) (*models.Organization, error) {
	return get[models.Organization](db, selectors...)
}

func ListOrganizations(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Organization, error) {
	return list[models.Organization](db, p, selectors...)
}

func DeleteOrganizations(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := GetOrganization(db, selectors...)
	if err != nil {
		return err
	}

	// TODO: delete everything in the organization

	return delete[models.Organization](db, toDelete.ID)
}

func UpdateOrganization(tx WriteTxn, org *models.Organization) error {
	return update(tx, (*organizationsTable)(org))
}
