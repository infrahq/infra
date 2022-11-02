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

func ByIDs(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id in (?)", ids)
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

func ByName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}

func ByIdentityID(identityID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("identity_id = ?", identityID)
	}
}

func ByPagination(p Pagination) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if p.Page == 0 && p.Limit == 0 {
			return db
		}
		resultsForPage := p.Limit * (p.Page - 1)
		return db.Offset(resultsForPage).Limit(p.Limit)
	}
}

func Limit(limit int) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	}
}

func NotName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not("name = ?", name)
	}
}

func NotPrivilege(privilege string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not("privilege = ?", privilege)
	}
}
