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

// func BindGroupGrants(db *gorm.DB, group *models.Group, grantIDs ...uid.ID) error {
// 	grants, err := ListGrants(db, ByIDs(grantIDs))
// 	if err != nil {
// 		return err
// 	}

// 	if err := db.Model(group).Association("Grants").Replace(grants); err != nil {
// 		return err
// 	}

// 	return nil
// }

func CreateGroup(db *gorm.DB, group *models.Group) error {
	return add(db, group)
}

// CreateOrUpdateGroup is deprecated
func CreateOrUpdateGroup(db *gorm.DB, group *models.Group, selectors ...SelectorFunc) (*models.Group, error) {
	existing, err := GetGroup(db, ByName(group.Name))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if err := CreateGroup(db, group); err != nil {
			return nil, err
		}

		return group, nil
	}

	if err := update(db, existing.ID, group); err != nil {
		return nil, err
	}

	return get[models.Group](db, ByID(existing.ID))
}

func GetGroup(db *gorm.DB, selectors ...SelectorFunc) (*models.Group, error) {
	return get[models.Group](db, selectors...)
}

func ListUserGroups(db *gorm.DB, userID uid.ID) (result []models.Group, err error) {
	err = db.Model("Group").Joins("User").Where("groups_users.user_id = ?", userID).Find(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func ListGroups(db *gorm.DB, selectors ...SelectorFunc) ([]models.Group, error) {
	return list[models.Group](db, selectors...)
}

func DeleteGroups(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return removeAll[models.Group](db, ByIDs(ids))
	}

	return nil
}
