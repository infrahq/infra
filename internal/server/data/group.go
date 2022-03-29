package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func BindGroupIdentities(db *gorm.DB, group *models.Group, identities ...models.Identity) error {
	if err := db.Model(group).Association("Identities").Replace(identities); err != nil {
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

func ListIdentityGroups(db *gorm.DB, userID uid.ID) (result []models.Group, err error) {
	user := &models.Identity{Model: models.Model{ID: userID}, Kind: models.UserKind}

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

		err := DeleteGrants(db, BySubject(g.PolyID()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Group](db, ByIDs(ids))
}
