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

func ByKeyID(key string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key_id = ?", key)
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

func ByOptionalIssuedFor(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if id == 0 {
			return db
		}

		return db.Where("issued_for = ?", id)
	}
}

func ByIssuedFor(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("issued_for = ?", id)
	}
}

func ByIdentityID(identityID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("identity_id = ?", identityID)
	}
}

func ByUserID(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
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
