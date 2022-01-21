package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(db *gorm.DB, destination *models.Destination) error {
	if err := add(db, &models.Destination{}, destination, &models.Destination{}); err != nil {
		return err
	}

	return nil
}

func UpdateDestination(db *gorm.DB, destination *models.Destination) error {
	if err := update(db, &models.Destination{}, destination, db.Where(destination, "id")); err != nil {
		return err
	}

	switch destination.Kind {
	case models.DestinationKindKubernetes:
		if err := db.Model(destination).Association("Kubernetes").Replace(&destination.Kubernetes); err != nil {
			return err
		}
	}

	if err := db.Model(destination).Association("Labels").Replace(&destination.Labels); err != nil {
		return err
	}

	return nil
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
	if err := list(db.Preload("Labels").Preload("Kubernetes"), &models.Destination{}, &destinations, condition); err != nil {
		return nil, err
	}

	return destinations, nil
}

func ListUserDestinations(db *gorm.DB, userID uid.ID) (result []models.Destination, err error) {
	var destinationIDs []uid.ID

	err = db.Model(models.Grant{}).Select("distinct destination_id").Joins("users_grants").Where("users_grants.user_id = ?", userID).Scan(&destinationIDs).Error
	if err != nil {
		return nil, err
	}

	err = db.Model(models.Destination{}).Where("id in (?)", destinationIDs).Find(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func DeleteDestinations(db *gorm.DB, selector SelectorFunc) error {
	toDelete, err := ListDestinations(db, selector(db))
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Destination{}, ids)
	}

	return internal.ErrNotFound
}
