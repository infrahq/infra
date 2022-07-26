package data

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func validateDestination(dest *models.Destination) error {
	if dest.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func CreateDestination(db *gorm.DB, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return add(db, destination)
}

func SaveDestination(db *gorm.DB, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return save(db, destination)
}

func GetDestination(db *gorm.DB, selectors ...SelectorFunc) (*models.Destination, error) {
	return get[models.Destination](db, selectors...)
}

func ListDestinations(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Destination, error) {
	return list[models.Destination](db, p, selectors...)
}

func DeleteDestinations(db *gorm.DB, selector SelectorFunc) error {
	toDelete, err := ListDestinations(db, nil, selector)
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

type destinationsCount struct {
	Connected bool
	Version   string
	Count     float64
}

func CountDestinationsByConnectedVersion(db *gorm.DB) ([]destinationsCount, error) {
	var results []destinationsCount
	timeout := time.Now().Add(-5 * time.Minute)
	if err := db.Raw("SELECT *, COUNT(*) AS count FROM (SELECT COALESCE(version, '') AS version, last_seen_at >= ? AS connected FROM destinations WHERE deleted_at IS NULL) AS d GROUP BY version, connected", timeout).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
