package data

import (
	"github.com/infrahq/infra/uid"
	"gorm.io/gorm"
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

func ByAPITokenIDs(id []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("api_token_id in (?)", id)
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

func ByIdentity(identity string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if identity == "" {
			return db
		}

		return db.Where("identity = ?", identity)
	}
}

func ByIdentityUserID(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if userID == 0 {
			return db
		}

		return db.Where("identity = ?", "u:"+userID.String())
	}
}

func ByIdentityGroupID(groupID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if groupID == 0 {
			return db
		}

		return db.Where("identity = ?", "g:"+groupID.String())
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
