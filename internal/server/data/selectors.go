package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/uid"
)

type SelectorFunc func(db *gorm.DB) *gorm.DB

func ByID(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func ByOrgID(orgID uid.ID) SelectorFunc {
	if orgID == 0 {
		panic("OrganizationID was not set")
	}
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("organization_id = ?", orgID)
	}
}
