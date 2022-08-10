package data

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

type orgCtxKey struct{}

// OrgFromContext returns the Organization stored using WithOrg.
func OrgFromContext(ctx context.Context) *models.Organization {
	org, ok := ctx.Value(orgCtxKey{}).(*models.Organization)
	if !ok {
		// TODO(orgs): panic("no org")
		return &models.Organization{}
	}
	return org
}

// WithOrg sets an Organization in the context. The Organization will be used
// by all query functions to insert,select, and modify entities within that
// organization.
func WithOrg(ctx context.Context, org *models.Organization) context.Context {
	return context.WithValue(ctx, orgCtxKey{}, org)
}

func CreateOrganization(db *gorm.DB, org *models.Organization) error {
	err := add(db, org)
	if err != nil {
		return fmt.Errorf("creating org: %w", err)
	}

	db.Statement.Context = WithOrg(db.Statement.Context, org)
	_, err = initializeSettings(db)
	if err != nil {
		return fmt.Errorf("initializing org settings: %w", err)
	}

	infraProvider := &models.Provider{
		Name:               models.InternalInfraProviderName,
		Kind:               models.ProviderKindInfra,
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
	}
	if err := CreateProvider(db, infraProvider); err != nil {
		return fmt.Errorf("failed to create infra provider: %w", err)
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
