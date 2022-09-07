package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
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

func NotIDs(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not(ids)
	}
}

func ByOptionalName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(name) > 0 {
			return db.Where("name = ?", name)
		}

		return db
	}
}

func ByOptionalIDs(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(ids) > 0 {
			return db.Where("id in (?)", ids)
		}

		return db
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

func ByOptionalUniqueID(nodeID string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(nodeID) > 0 {
			return db.Where("unique_id = ?", nodeID)
		}

		return db
	}
}

func ByProviderID(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("provider_id = ?", id)
	}
}

func ByOptionalSubject(polymorphicID uid.PolymorphicID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if polymorphicID == "" {
			return db
		}

		return db.Where("subject = ?", string(polymorphicID))
	}
}

func BySubject(polymorphicID uid.PolymorphicID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("subject = ?", string(polymorphicID))
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

func CreatedBy(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("created_by = ?", id)
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

func NotProviderKind(kind models.ProviderKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not("kind = ?", kind)
	}
}

func ByProviderKind(kind models.ProviderKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("kind = ?", kind)
	}
}

func NotPrivilege(privilege string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not("privilege = ?", privilege)
	}
}

func NotInfraConnector() SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(`NOT (privilege = 'connector' AND resource = 'infra')`)
	}
}

func ByOptionalIdentityGroupID(groupID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if groupID == 0 {
			return db
		}
		return db.
			Joins("join identities_groups on identities_groups.identity_id = id").
			Where("identities_groups.group_id = ?", groupID)
	}
}

func Preload(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Preload(name)
	}
}

func ByDomain(host string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("domain = ?", host)
	}
}
