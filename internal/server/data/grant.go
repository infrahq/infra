package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) error {
	// check first if it exists
	grants, err := list[models.Grant](db, BySubject(grant.Subject), ByResource(grant.Resource))
	if err != nil {
		return err
	}

	for _, existingGrant := range grants {
		if existingGrant.Privilege == grant.Privilege {
			// exact match exists, no need to store it twice.
			return nil
		}
	}

	return add(db, grant)
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListGrants(db *gorm.DB, selectors ...SelectorFunc) ([]models.Grant, error) {
	return list[models.Grant](db, selectors...)
}

func DeleteGrants(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.Grant](db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, g := range toDelete {
		ids = append(ids, g.ID)
	}

	return deleteAll[models.Grant](db, ByIDs(ids))
}

func ByOptionalPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("privilege = ?", s)
	}
}

func GrantsInheritedByUser(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		var groupIDs []uid.ID
		err := db.Raw("select distinct group_id from identities_groups where identity_id = ?", userID).Pluck("group_id", &groupIDs).Error
		if err != nil {
			logging.S.Errorf("GrantsInheritedByUser: %s", err)
		}
		subjects := []string{uid.NewIdentityPolymorphicID(userID).String()}
		for _, groupID := range groupIDs {
			subjects = append(subjects, uid.NewGroupPolymorphicID(groupID).String())
		}
		return db.Where("subject in (?)", subjects)
	}
}

func ByPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("privilege = ?", s)
	}
}

func ByOptionalResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("resource = ?", s)
	}
}

func ByResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("resource = ?", s)
	}
}
