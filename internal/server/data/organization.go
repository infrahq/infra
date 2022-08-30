package data

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// CreateOrganization creates a new organization and sets the current db context to execute on this org
func CreateOrganization(tx GormTxn, org *models.Organization) error {
	err := add(tx, org)
	if err != nil {
		return fmt.Errorf("creating org: %w", err)
	}

	tx = NewTransaction(tx.GormDB(), org.ID)

	_, err = initializeSettings(tx)
	if err != nil {
		return fmt.Errorf("initializing org settings: %w", err)
	}

	infraProvider := &models.Provider{
		Name:      models.InternalInfraProviderName,
		Kind:      models.ProviderKindInfra,
		CreatedBy: models.CreatedBySystem,
	}
	if err := CreateProvider(tx, infraProvider); err != nil {
		return fmt.Errorf("failed to create infra provider: %w", err)
	}

	connector := &models.Identity{
		Name:      models.InternalInfraConnectorIdentityName,
		CreatedBy: models.CreatedBySystem,
	}
	// this identity is used to create access keys for connectors
	if err := CreateIdentity(tx, connector); err != nil {
		return fmt.Errorf("failed to create connector identity while creating org: %w", err)
	}

	err = CreateGrant(tx, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(connector.ID),
		Privilege: models.InfraConnectorRole,
		Resource:  "infra",
		CreatedBy: models.CreatedBySystem,
	})
	if err != nil {
		return fmt.Errorf("failed to grant connector role creating org: %w", err)
	}

	return nil
}

func GetOrganization(db GormTxn, selectors ...SelectorFunc) (*models.Organization, error) {
	return get[models.Organization](db, selectors...)
}

func ListOrganizations(db GormTxn, p *models.Pagination, selectors ...SelectorFunc) ([]models.Organization, error) {
	return list[models.Organization](db, p, selectors...)
}

func DeleteOrganizations(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := GetOrganization(db, selectors...)
	if err != nil {
		return err
	}

	// TODO:
	//   * Delete grants
	//   * Delete groups
	//   * Delete users

	return delete[models.Organization](db, toDelete.ID)
}

func UpdateOrganization(tx GormTxn, org *models.Organization) error {
	return save(tx, org)
}
