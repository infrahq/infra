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
		if existingGrant.Privilege == grant.Privilege &&
			existingGrant.ExpiresAfterUnused == grant.ExpiresAfterUnused &&
			existingGrant.ExpiresAt == grant.ExpiresAt {
			// exact match exists, no need to store it twice.
			return nil
		}
	}

	return add(db, grant)
}

func GetGrant(db *gorm.DB, selectors ...SelectorFunc) (*models.Grant, error) {
	return get[models.Grant](db, selectors...)
}

func ListUserGrants(db *gorm.DB, userID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewUserPolymorphicID(userID)
	return ListGrants(db, BySubject(polymorphicID), NotCreatedBy(models.CreatedBySystem))
}

func ListMachineGrants(db *gorm.DB, machineID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewMachinePolymorphicID(machineID)
	return ListGrants(db, BySubject(polymorphicID), NotCreatedBy(models.CreatedBySystem))
}

func ListGroupGrants(db *gorm.DB, groupID uid.ID) (result []models.Grant, err error) {
	polymorphicID := uid.NewGroupPolymorphicID(groupID)
	return ListGrants(db, BySubject(polymorphicID), NotCreatedBy(models.CreatedBySystem))
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

		return db.Where("resource = ?", s)
	}
}
