package data

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGrant(db GormTxn, grant *models.Grant) error {
	switch {
	case grant.Subject == "":
		return fmt.Errorf("subject is required")
	case grant.Privilege == "":
		return fmt.Errorf("privilege is required")
	case grant.Resource == "":
		return fmt.Errorf("resource is required")
	}
	return add(db, grant)
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListGrants(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Grant, error) {
	return list[models.Grant](db, p, selectors...)
}

func DeleteGrants(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := list[models.Grant](db, nil, selectors...)
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

func GrantsInheritedBySubject(subjectID uid.PolymorphicID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		switch {
		case subjectID.IsIdentity():
			userID, err := subjectID.ID()
			if err != nil {
				logging.Errorf("invalid subject id %q", subjectID)
				return db.Where("1 = 0")
			}
			var groupIDs []uid.ID
			err = db.Session(&gorm.Session{NewDB: true}).Raw("select distinct group_id from identities_groups where identity_id = ?", userID).Pluck("group_id", &groupIDs).Error
			if err != nil {
				logging.Errorf("GrantsInheritedByUser: %s", err)
				_ = db.AddError(err)
				return db.Where("1 = 0")
			}

			subjects := []string{subjectID.String()}
			for _, groupID := range groupIDs {
				subjects = append(subjects, uid.NewGroupPolymorphicID(groupID).String())
			}
			return db.Where("subject in (?)", subjects)
		case subjectID.IsGroup():
			return BySubject(subjectID)(db)
		default:
			panic("unhandled subject type")
		}
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
