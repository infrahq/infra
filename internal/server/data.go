package server

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Email    string `json:"email" gorm:"unique"`
	Password []byte `json:"-"`
	Provider string `json:"provider"`
	Created  int64  `json:"created" gorm:"autoCreateTime"`
	Updated  int64  `json:"updated" gorm:"autoUpdateTime"`
}

type Settings struct {
	ID          uint  `gorm:"primaryKey"`
	Created     int64 `json:"created" gorm:"autoCreateTime"`
	Updated     int64 `json:"updated" gorm:"autoUpdateTime"`
	TokenSecret []byte
}

func NewDB(dbpath string) (*gorm.DB, error) {
	if err := os.MkdirAll(dbpath, os.ModePerm); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(dbpath, "infra.db")), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Settings{})

	return db, nil
}

func SyncUsers(db *gorm.DB, emails []string, provider string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, email := range emails {
			var user User
			tx.Where("email = ?", email).First(&user)

			user.Email = email
			user.Provider = "okta"

			if err := tx.Save(&user).Error; err != nil {
				return err
			}

			var users []User
			if err := tx.Find(&users).Error; err != nil {
				return err
			}

			emailsMap := make(map[string]bool)
			for _, email := range emails {
				emailsMap[email] = true
			}

			for _, user := range users {
				if !emailsMap[user.Email] {

					// Only delete user if they don't have built in auth
					// TODO (jmorganca): properly refactor this into a provider check
					if user.Provider == "okta" && len(user.Password) == 0 {
						if err := tx.Delete(&user).Error; err != nil {
							return err
						}
					} else {
						user.Provider = "infra"
						if err := tx.Save(&user).Error; err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	})
}
