package data

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func BindUserGroups(db *gorm.DB, user *models.User, groups ...models.Group) error {
	if err := db.Model(user).Association("Groups").Replace(groups); err != nil {
		return fmt.Errorf("bind user groups: %w", err)
	}

	return nil
}

func CreateUser(db *gorm.DB, user *models.User) error {
	return add(db, user)
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

	ids := make([]uid.ID, 0)
	for _, u := range toDelete {
		ids = append(ids, u.ID)

		err := DeleteGrants(db, ByIdentity(u.PolymorphicIdentifier()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.User](db, ByIDs(ids))
}

func SaveUser(db *gorm.DB, user *models.User) error {
	return save(db, user)
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
