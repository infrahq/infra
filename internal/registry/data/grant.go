package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) (*models.Grant, error) {
	if err := add(db, &models.Grant{}, grant, &models.Grant{}); err != nil {
		return nil, err
	}

	return grant, nil
}

func CreateOrUpdateGrant(db *gorm.DB, grant *models.Grant, condition interface{}) (*models.Grant, error) {
	existing, err := GetGrant(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateGrant(db, grant); err != nil {
			return nil, err
		}

		return grant, nil
	}

	if err := update(db, &models.Grant{}, grant, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch grant.Kind {
	case models.GrantKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&grant.Kubernetes); err != nil {
			return nil, err
		}
	}

	return GetGrant(db, db.Where(existing, "id"))
}

func GetGrant(db *gorm.DB, condition interface{}) (*models.Grant, error) {
	var grant models.Grant
	if err := get(db, &models.Grant{}, &grant, condition); err != nil {
		return nil, err
	}

	return &grant, nil
}

func ListGrants(db *gorm.DB, condition interface{}) ([]models.Grant, error) {
	grants := make([]models.Grant, 0)
	if err := list(db, &models.Grant{}, &grants, condition); err != nil {
		return nil, err
	}

	return grants, nil
}

func DeleteGrants(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListGrants(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return delete(db, &models.Grant{}, ids)
	}

	return nil
}

func GrantSelector(db *gorm.DB, grant *models.Grant) *gorm.DB {
	switch grant.Kind {
	case models.GrantKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&models.GrantKubernetes{}).Select("grant_id").Where(grant.Kubernetes),
		)
	}

	if grant.Destination.ID != uuid.Nil {
		db = db.Where("destination_id in (?)", grant.Destination.ID)
	}

	return db.Where(&grant)
}

// StrictGrantSelector matches all fields exactly, including initialized fields.
func StrictGrantSelector(db *gorm.DB, grant *models.Grant) *gorm.DB {
	switch grant.Kind {
	case models.GrantKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&models.GrantKubernetes{}).Select("grant_id").Where(map[string]interface{}{
				"Kind":      grant.Kubernetes.Kind,
				"Name":      grant.Kubernetes.Name,
				"Namespace": grant.Kubernetes.Namespace,
			}),
		)
	}

	if grant.Destination.ID != uuid.Nil {
		db = db.Where("destination_id in (?)", grant.Destination.ID)
	}

	return db.Where(&grant)
}
