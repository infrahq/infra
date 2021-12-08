package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateDestination(db *gorm.DB, destination *models.Destination) (*models.Destination, error) {
	if err := add(db, &models.Destination{}, destination, &models.Destination{}); err != nil {
		return nil, err
	}

	return destination, nil
}

func CreateOrUpdateDestination(db *gorm.DB, destination *models.Destination, condition interface{}) (*models.Destination, error) {
	existing, err := GetDestination(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateDestination(db, destination); err != nil {
			return nil, err
		}

		return destination, nil
	}

	if err := update(db, &models.Destination{}, destination, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch destination.Kind {
	case models.DestinationKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&destination.Kubernetes); err != nil {
			return nil, err
		}
	}

	if err := db.Model(existing).Association("Labels").Replace(&destination.Labels); err != nil {
		return nil, err
	}

	return GetDestination(db, db.Where(existing, "id"))
}

func GetDestination(db *gorm.DB, condition interface{}) (*models.Destination, error) {
	var destination models.Destination
	if err := get(db, &models.Destination{}, &destination, condition); err != nil {
		return nil, err
	}

	return &destination, nil
}

func ListDestinations(db *gorm.DB, condition interface{}) ([]models.Destination, error) {
	destinations := make([]models.Destination, 0)
	if err := list(db, &models.Destination{}, &destinations, condition); err != nil {
		return nil, err
	}

	return destinations, nil
}

func UpdateDestination(db *gorm.DB, destination *models.Destination, selector SelectorFunc) (*models.Destination, error) {
	existing, err := GetDestination(db, selector(db))
	if err != nil {
		return nil, err
	}

	if err := update(db, &models.Destination{}, destination, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch destination.Kind {
	case models.DestinationKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&destination.Kubernetes); err != nil {
			return nil, err
		}
	}

	if err := db.Model(existing).Association("Labels").Replace(&destination.Labels); err != nil {
		return nil, err
	}

	return GetDestination(db, db.Where(existing, "id"))
}

func DeleteDestinations(db *gorm.DB, selector SelectorFunc) error {
	toDelete, err := ListDestinations(db, selector(db))
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Destination{}, ids)
	}

	return internal.ErrNotFound
}
