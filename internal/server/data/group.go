package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func BindGroupUsers(db *gorm.DB, group *models.Group, users ...models.User) error {
	if err := db.Model(group).Association("Users").Replace(users); err != nil {
		return err
	}

	return nil
}

func CreateGroup(db *gorm.DB, group *models.Group) error {
	return add(db, group)
}

func GetGroup(db *gorm.DB, selectors ...SelectorFunc) (*models.Group, error) {
	return get[models.Group](db, selectors...)
}

func ListGroups(db *gorm.DB, selectors ...SelectorFunc) ([]models.Group, error) {
	return list[models.Group](db, selectors...)
}

func ListUserGroups(db *gorm.DB, userID uid.ID) (result []models.Group, err error) {
	user := &models.User{Model: models.Model{ID: userID}}

	if err := db.Model(user).Association("Groups").Find(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func DeleteGroups(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListGroups(db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, g := range toDelete {
		ids = append(ids, g.ID)

		err := DeleteGrants(db, ByIdentity(g.PolymorphicIdentifier()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}
