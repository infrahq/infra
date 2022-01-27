package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func CreateDestination(db *gorm.DB, destination *models.Destination) error {
	return add(db, destination)
}

func UpdateDestination(db *gorm.DB, destination *models.Destination) error {
	if err := save(db, destination); err != nil {
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

func GetDestination(db *gorm.DB, selectors ...SelectorFunc) (*models.Destination, error) {
	return get[models.Destination](db, selectors...)
}

func ListDestinations(db *gorm.DB, selectors ...SelectorFunc) ([]models.Destination, error) {
	return list[models.Destination](db.Preload("Labels").Preload("Kubernetes"), selectors...)
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
	toDelete, err := ListDestinations(db, selector)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return removeAll[models.Destination](db, ByIDs(ids))
	}

	return internal.ErrNotFound
}
