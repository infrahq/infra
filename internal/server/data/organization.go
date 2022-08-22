package data

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type orgCtxKey struct{}

// OrgFromContext returns the Organization stored using WithOrg.
func OrgFromContext(ctx context.Context) *models.Organization {
	org, ok := ctx.Value(orgCtxKey{}).(*models.Organization)
	if !ok {
		return nil
	}
	return org
}

func MustGetOrgFromContext(ctx context.Context) *models.Organization {
	org := OrgFromContext(ctx)
	if org == nil {
		panic("no organization found in context. this should never happen")
	}
	return org
}

// WithOrg sets an Organization in the context. The Organization will be used
// by all query functions to insert, select, and modify entities within that
// organization.
func WithOrg(ctx context.Context, org *models.Organization) context.Context {
	return context.WithValue(ctx, orgCtxKey{}, org)
}

// CreateOrganizationAndSetContext creates a new organization and sets the current db context to execute on this org
func CreateOrganizationAndSetContext(tx GormTxn, org *models.Organization) error {
	err := add(tx, org)
	if err != nil {
		return fmt.Errorf("creating org: %w", err)
	}

	db := tx.GormDB()
	// TODO: remove
	db.Statement.Context = WithOrg(db.Statement.Context, org)
	// TODO: constructor?
	tx = &Transaction{DB: db, orgID: org.ID}

	_, err = initializeSettings(db)
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
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
		CreatedBy: models.CreatedBySystem,
	})
	if err != nil {
		return fmt.Errorf("failed to grant connector role creating org: %w", err)
	}

	return nil
}

func GetOrganization(db *gorm.DB, selectors ...SelectorFunc) (*models.Organization, error) {
	return get[models.Organization](db, selectors...)
}

func ListOrganizations(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Organization, error) {
	return list[models.Organization](db, p, selectors...)
}

func DeleteOrganizations(db *gorm.DB, selectors ...SelectorFunc) error {
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
