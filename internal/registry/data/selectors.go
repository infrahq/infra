package data

import "gorm.io/gorm"

type SelectorFunc func(db *gorm.DB) *gorm.DB

func ByID(id string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func ByName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}
