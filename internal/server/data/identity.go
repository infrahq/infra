package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func BindIdentityGroups(db *gorm.DB, identity *models.Identity, groups ...models.Group) error {
	return bindAssociations(db, identity, "Groups", groups)
}

func CreateIdentity(db *gorm.DB, identity *models.Identity) error {
	return add(db, identity)
}

func GetIdentity(db *gorm.DB, selectors ...SelectorFunc) (*models.Identity, error) {
	return get[models.Identity](db, selectors...)
}

func ListIdentities(db *gorm.DB, selectors ...SelectorFunc) ([]models.Identity, error) {
	return list[models.Identity](db, selectors...)
}

func DeleteIdentity(db *gorm.DB, id uid.ID) error {
	return delete[models.Identity](db, id)
}

func DeleteIdentities(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListIdentities(db.Select("id"), selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, i := range toDelete {
		ids = append(ids, i.ID)

		err := DeleteGrants(db, BySubject(i.PolyID()))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Identity](db, ByIDs(ids))
}

func SaveIdentity(db *gorm.DB, identity *models.Identity) error {
	return save(db, identity)
}
