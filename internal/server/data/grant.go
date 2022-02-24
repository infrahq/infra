package data

import (
	"strings"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) error {
	return add(db, grant)
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListUserGrants(db *gorm.DB, userID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewUserPolymorphicID(userID)
	return ListGrants(db, ByIdentity(polymorphicID), NotCreatedBySystem())
}

func ListMachineGrants(db *gorm.DB, machineID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewMachinePolymorphicID(machineID)
	return ListGrants(db, ByIdentity(polymorphicID), NotCreatedBySystem())
}

func ListGroupGrants(db *gorm.DB, groupID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewGroupPolymorphicID(groupID)
	return ListGrants(db, ByIdentity(polymorphicID), NotCreatedBySystem())
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

func ByPrivilege(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		return db.Where("privilege = ?", s)
	}
}

func ByResource(s string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if s == "" {
			return db
		}

		// remove the trailing resource name if it's contained in a parent
		split := strings.Split(s, ".")
		if len(split) > 1 {
			split = split[:len(split)-1]
			s = strings.Join(split, ".")
		}

		// match anything contained in this resource
		s = s + "%"

		return db.Where("resource LIKE (?)", s)
	}
}
