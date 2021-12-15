package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) (*models.Grant, error) {
	if err := FindOrInitLabels(db, grant.Labels); err != nil {
		return nil, err
	}

	if err := add(db, &models.Grant{}, grant, &models.Grant{}); err != nil {
		return nil, err
	}

	return grant, nil
}

func CreateOrUpdateGrant(db *gorm.DB, grant *models.Grant, condition interface{}) (*models.Grant, error) {
	if err := FindOrInitLabels(db, grant.Labels); err != nil {
		return nil, err
	}

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

	return GetGrant(db, db.Where(existing, "id"))
}

func GetGrant(db *gorm.DB, condition interface{}) (*models.Grant, error) {
	var grant models.Grant
	if err := get(db, &models.Grant{}, &grant, condition); err != nil {
		return nil, err
	}

	return &grant, nil
}

func ListGrants(db *gorm.DB, selector SelectorFunc) ([]models.Grant, error) {
	grants := make([]models.Grant, 0)
	if err := list(db, &models.Grant{}, &grants, selector(db)); err != nil {
		return nil, err
	}

	return grants, nil
}

func UpdateGrant(db *gorm.DB, grant *models.Grant, selector SelectorFunc) (*models.Grant, error) {
	if err := FindOrInitLabels(db, grant.Labels); err != nil {
		return nil, err
	}

	existing, err := GetGrant(db, selector(db))
	if err != nil {
		return nil, err
	}

	if err := update(db, &models.Grant{}, grant, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	return GetGrant(db, db.Where(existing, "id"))
}

func DeleteGrants(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListGrants(db, func(db *gorm.DB) *gorm.DB {
		return db.Where(condition)
	})
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Grant{}, ids)
	}

	return nil
}

func ByGrantUser(user *models.User) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins("Users", user)
	}
}

func ByGrantGroup(group *models.Group) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins("Groups", group)
	}
}

func ByGrantDestination(destination *models.Destination) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		db = db.Where("resource_kind = ?", destination.Kind)
		db = db.Where("resource_name = ?", destination.Name).Or("resource_name = ''")
		db = db.Joins("Labels", destination.Labels)
		return db
	}
}
