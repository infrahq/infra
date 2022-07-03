package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	//"github.com/infrahq/infra/uid"
)

func CreateOrganization(db *gorm.DB, org *models.Organization) error {
	return add(db, org)
}

func GetOrganization(db *gorm.DB, selectors ...SelectorFunc) (*models.Organization, error) {
	return get[models.Organization](db, selectors...)
}

func ListOrganizations(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Organization, error) {
	return list[models.Organization](db, p, selectors...)
}

func DeleteOrganization(db *gorm.DB, selectors ...SelectorFunc) error {
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
