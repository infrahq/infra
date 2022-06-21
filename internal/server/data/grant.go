package data

import (
	"gorm.io/gorm"

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
		if sameGrant(&existingGrant, grant) {
			// exact match exists, errors
			return UniqueConstraintError{Table: "grants", Column: "subject, resource, and privilege"}
		}
	}

	return add(db, grant)
}

func sameGrant(grant1, grant2 *models.Grant) bool {
	return grant1.Subject == grant2.Subject && grant1.Resource == grant2.Resource && grant1.Privilege == grant2.Privilege
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
