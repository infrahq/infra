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

func ByName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(name) > 0 {
			return db.Where("name = ?", name)
		}

		return db
	}
}

func ByUniqueID(nodeID string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(nodeID) > 0 {
			return db.Where("unique_id = ?", nodeID)
		}

		return db
	}
}

func ByEmail(email string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(email) > 0 {
			return db.Where("email = ?", email)
		}

		return db
	}
}

func ByProviderID(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if id == 0 {
			return db
		}

		return db.Where("provider_id = ?", id)
	}
}

func ByKey(key string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
}

func ByURL(url string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(url) == 0 {
			return db
		}

		return db.Where("url = ?", url)
	}
}

func ByIdentity(polymorphicID uid.PolymorphicID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if polymorphicID == "" {
			return db
		}

		return db.Where("identity = ?", string(polymorphicID))
	}
}

func ByMachineIDIssuedFor(machineID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if machineID == 0 {
			return db
		}

		return db.Where("issued_for = ?", uid.NewMachinePolymorphicID(machineID))
	}
}

func ByUserIDIssuedFor(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if userID == 0 {
			return db
		}

		return db.Where("issued_for = ?", uid.NewUserPolymorphicID(userID))
	}
}

func ByUserID(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if userID == 0 {
			return db
		}

		return db.Where("user_id = ?", userID)
	}
}

// NotCreatedBySystem filters out any entities that do not have a "created by" field set, meaning they were created internally
func NotCreatedBySystem() SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		// the created_by field is default 0 when not set by default
		return db.Where("created_by != 0")
	}
}
