package data

import (
	"fmt"
	"time"

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

func CreateDestination(db GormTxn, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return add(db, destination)
}

func SaveDestination(db GormTxn, destination *models.Destination) error {
	if err := validateDestination(destination); err != nil {
		return err
	}
	return save(db, destination)
}

func GetDestination(db GormTxn, selectors ...SelectorFunc) (*models.Destination, error) {
	return get[models.Destination](db, selectors...)
}

func ListDestinations(db GormTxn, p *Pagination, selectors ...SelectorFunc) ([]models.Destination, error) {
	return list[models.Destination](db, p, selectors...)
}

func DeleteDestinations(db GormTxn, selector SelectorFunc) error {
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

func CountDestinationsByConnectedVersion(tx ReadTxn) ([]destinationsCount, error) {
	timeout := time.Now().Add(-5 * time.Minute)

	stmt := `
		SELECT *, COUNT(*) AS count
		FROM (
			SELECT COALESCE(version, '') AS version, last_seen_at >= ? AS connected
			FROM destinations
			WHERE deleted_at IS NULL
		) AS d
		GROUP BY version, connected`
	rows, err := tx.Query(stmt, timeout)
	if err != nil {
		return nil, err

	}
	defer rows.Close()

	var result []destinationsCount
	for rows.Next() {
		var item destinationsCount
		if err := rows.Scan(&item.Version, &item.Connected, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}
