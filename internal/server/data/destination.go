package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(db *gorm.DB, destination *models.Destination) error {
	return add(db, destination)
}

func SaveDestination(db *gorm.DB, destination *models.Destination) error {
	if err := save(db, destination); err != nil {
		return err
	}

	return nil
}

func GetDestination(db *gorm.DB, selectors ...SelectorFunc) (*models.Destination, error) {
	return get[models.Destination](db, selectors...)
}

func ListDestinations(db *gorm.DB, selectors ...SelectorFunc) ([]models.Destination, error) {
	return list[models.Destination](db, selectors...)
}

func DeleteDestinations(db *gorm.DB, selector SelectorFunc) error {
	toDelete, err := ListDestinations(db, selector)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return deleteAll[models.Destination](db, ByIDs(ids))
	}

	return internal.ErrNotFound
}
