package data

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func BindUserGroups(db *gorm.DB, user *models.User, groups ...models.Group) error {
	if err := db.Model(user).Association("Groups").Replace(groups); err != nil {
		return fmt.Errorf("bind user groups: %w", err)
	}

	return nil
}

// func BindUserGrants(db *gorm.DB, user *models.User, grantIDs ...uid.ID) error {
// 	grants, err := ListGrants(db, ByIDs(grantIDs))
// 	if err != nil {
// 		return err
// 	}

// 	if err := db.Model(user).Association("Grants").Replace(grants); err != nil {
// 		return err
// 	}

// 	return nil
// }

func CreateUser(db *gorm.DB, user *models.User) error {
	return add(db, user)
}

// CreateOrUpdateUser is deprecated
func CreateOrUpdateUser(db *gorm.DB, user *models.User, selectors ...SelectorFunc) (*models.User, error) {
	existing, err := GetUser(db, ByEmail(user.Email))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if err := CreateUser(db, user); err != nil {
			return nil, err
		}

		return user, nil
	}

	if err := UpdateUser(db, user, ByID(existing.ID)); err != nil {
		return nil, err
	}

	return GetUser(db, ByID(existing.ID))
}

func GetUser(db *gorm.DB, selectors ...SelectorFunc) (*models.User, error) {
	return get[models.User](db, selectors...)
}

func ListUsers(db *gorm.DB, selectors ...SelectorFunc) ([]models.User, error) {
	return list[models.User](db, selectors...)
}

func DeleteUsers(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListUsers(db.Select("id"), selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return removeAll[models.User](db, ByIDs(ids))
	}

	return nil
}

func UpdateUser(db *gorm.DB, user *models.User, selector SelectorFunc) error {
	existing, err := GetUser(db, selector)
	if err != nil {
		return fmt.Errorf("get existing: %w", err)
	}

	if err := save(db, user); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if err := db.Model(existing).Association("Grants").Replace(&user.Grants); err != nil {
		return fmt.Errorf("grants: %w", err)
	}

	if err := db.Model(existing).Association("Providers").Replace(&user.Providers); err != nil {
		return fmt.Errorf("providers: %w", err)
	}

	if err := db.Model(existing).Association("Groups").Replace(&user.Groups); err != nil {
		return fmt.Errorf("groups: %w", err)
	}

	return nil
}

func UserAssociations(db *gorm.DB) *gorm.DB {
	db = db.Preload("Grants.Kubernetes").Preload("Grants.Destination.Kubernetes")
	db = db.Preload("Groups.Grants.Kubernetes").Preload("Groups.Grants.Destination.Kubernetes")

	return db
}

func ByEmailInList(emails []string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("email in (?)", emails)
	}
}

func ByIDNotInList(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(ids) > 0 {
			return db.Where("id not in (?)", ids)
		}

		return db
	}
}
