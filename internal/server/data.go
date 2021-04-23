package server

import (
	"github.com/rs/xid"
	"gorm.io/gorm"
)

const (
	USER_ID_PREFIX = "usr"
)

type User struct {
	ID             string `gorm:"primaryKey" json:"id"`
	Username       string `json:"username"`
	HashedPassword []byte `json:"-"`
	Created        int    `gorm:"autoCreateTime" json:"created"`
	Updated        int    `gorm:"autoUpdateTime" json:"updated"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = USER_ID_PREFIX + xid.New().String()
	return nil
}
