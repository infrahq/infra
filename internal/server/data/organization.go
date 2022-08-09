package data

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

type OrgCtxKey struct{}

func CreateOrganization(db *gorm.DB, org *models.Organization) error {
	err := add(db, org)
	if err != nil {
		return fmt.Errorf("creating org: %w", err)
	}

	_, err = InitializeSettings(db, org)
	if err != nil {
		return fmt.Errorf("initializing org settings: %w", err)
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
