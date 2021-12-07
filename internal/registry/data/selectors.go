package data

import "gorm.io/gorm"

type SelectorFunc func(db *gorm.DB) *gorm.DB

func ByName(name string) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", name)
	}
}
