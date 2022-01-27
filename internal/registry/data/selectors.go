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

func ByAPITokenID(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("api_token_id = ?", id)
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

func ByNodeID(nodeID string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(nodeID) > 0 {
			return db.Where("node_id = ?", nodeID)
		}

		return db
	}
}

func ByNameInList(names []string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name in (?)", names)
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

func ByIDs(ids []uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id in (?)", ids)
	}
}

func ByKey(key string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
}

func ByDestinationID(id uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if id == 0 {
			return db
		}

		return db.Where("destination_id = ?", id)
	}
}

func ByUserID(userID uid.ID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}
