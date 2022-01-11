package data

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SelectorFunc func(db *gorm.DB) *gorm.DB

func ByID(id string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func ByUUID(id uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func ByName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}
