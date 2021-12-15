package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func BindGroupUsers(db *gorm.DB, group *models.Group, users ...models.User) error {
	if err := db.Model(group).Association("Users").Replace(users); err != nil {
		return err
	}

	return nil
}

func BindGroupGrants(db *gorm.DB, group *models.Group, grantIDs ...uuid.UUID) error {
	grants, err := ListGrants(db, grantIDs)
	if err != nil {
		return err
	}

	if err := db.Model(group).Association("Grants").Replace(grants); err != nil {
		return err
	}

	return nil
}

func CreateGroup(db *gorm.DB, group *models.Group) (*models.Group, error) {
	if err := add(db, &models.Group{}, group, &models.Group{}); err != nil {
		return nil, err
	}

	return group, nil
}

func CreateOrUpdateGroup(db *gorm.DB, group *models.Group, condition interface{}) (*models.Group, error) {
	existing, err := GetGroup(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateGroup(db, group); err != nil {
			return nil, err
		}

		return group, nil
	}

	if err := update(db, &models.Group{}, group, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	if err := get(db, &models.Group{}, group, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	return group, nil
}

func GetGroup(db *gorm.DB, condition interface{}) (*models.Group, error) {
	var group models.Group
	if err := get(db, &models.Group{}, &group, condition); err != nil {
		return nil, err
	}

	return &group, nil
}

func ListGroups(db *gorm.DB, condition interface{}) ([]models.Group, error) {
	groups := make([]models.Group, 0)
	if err := list(db, &models.Group{}, &groups, condition); err != nil {
		return nil, err
	}

	return groups, nil
}

func DeleteGroups(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListGroups(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Group{}, ids)
	}

	return nil
}

func GroupAssociations(db *gorm.DB) *gorm.DB {
	return db
}
