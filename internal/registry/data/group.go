package data

import (
	"errors"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func BindGroupUsers(db *gorm.DB, group *models.Group, users ...models.User) error {
	if err := db.Model(group).Association("Users").Replace(users); err != nil {
		return err
	}

	return nil
}

func BindGroupGrants(db *gorm.DB, group *models.Group, grantIDs ...uid.ID) error {
	grants, err := ListGrants(db, ByIDs(grantIDs))
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

func ListUserGroups(db *gorm.DB, userID uid.ID) (result []models.Group, err error) {
	err = db.Model("Group").Joins("User").Where("groups_users.user_id = ?", userID).Find(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
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
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Group{}, ids)
	}

	return nil
}

func GroupAssociations(db *gorm.DB) *gorm.DB {
	db = db.Preload("Grants.Kubernetes").Preload("Grants.Destination.Kubernetes")
	db = db.Preload("Users.Grants.Kubernetes").Preload("Users.Grants.Destination.Kubernetes")

	return db
}
