package data

import (
	"github.com/infrahq/infra/uuid"
	"gorm.io/gorm"
)

type SelectorFunc func(db *gorm.DB) *gorm.DB

func ByID(id uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func ByAPITokenID(id uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("api_token_id = ?", id)
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

func ByEmail(email string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(email) > 0 {
			return db.Where("email = ?", email)
		}

		return db
	}
}

func ByIDs(ids []uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id in (?)", ids)
	}
}

func ByKey(key string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
}

func ByDestinationID(id uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if id == 0 {
			return db
		}

		return db.Where("destination_id = ?", id)
	}
}
