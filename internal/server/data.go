package server

import (
	"errors"
	"os"
	"path"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Created  int64  `json:"created" gorm:"autoCreateTime"`
	Updated  int64  `json:"updated" gorm:"autoUpdateTime"`
	Email    string `json:"email" gorm:"unique"`
	Password []byte `json:"-"`
	Provider string `json:"provider"`

	Permission   Permission `json:"permission"`
	PermissionID uint       `json:"-"`
}

type Permission struct {
	ID          uint   `json:"id" yaml:"-" gorm:"primaryKey"`
	Created     int64  `json:"created" yaml:"-" gorm:"autoCreateTime"`
	Updated     int64  `json:"updated" yaml:"-" gorm:"autoUpdateTime"`
	Name        string `json:"name" yaml:"name" gorm:"unique"`
	Description string `json:"description" yaml:"description"`

	KubernetesRole      string `json:"kubernetesRole" yaml:"kubernetesRole"`
	KubernetesNamespace string `json:"kubernetesNamespace" yaml:"kubernetesNamespace"`

	Users []User `json:"-"`
}

type Settings struct {
	ID      uint   `gorm:"primaryKey"`
	Created int64  `json:"created" gorm:"autoCreateTime"`
	Updated int64  `json:"updated" gorm:"autoUpdateTime"`
	Config  []byte `json:"config"`
}

var DefaultPermissions = []Permission{
	{
		Name:           "view",
		Description:    "Read most resources",
		KubernetesRole: "view",
	},
	{
		Name:           "edit",
		Description:    "Read & write most resources",
		KubernetesRole: "edit",
	},
	{
		Name:           "admin",
		Description:    "Read & write all resources",
		KubernetesRole: "admin",
	},
}

func NewDB(dbpath string) (*gorm.DB, error) {
	if err := os.MkdirAll(path.Dir(dbpath), os.ModePerm); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Permission{})
	db.AutoMigrate(&Settings{})

	// Add default permissions
	for _, p := range DefaultPermissions {
		err := db.Where(&Permission{Name: p.Name}).First(&p).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		db.Save(&p)
	}

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

func IsEqualOrHigherPermission(a string, b string) bool {
	indexa := 0
	indexb := 0

	for i, p := range DefaultPermissions {
		if a == p.Name {
			indexa = i
		}

		if b == p.Name {
			indexb = i
		}
	}

	return indexa >= indexb
}
