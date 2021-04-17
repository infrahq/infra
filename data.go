package main

import (
	"github.com/rs/xid"
	"gorm.io/gorm"
)

type User struct {
	ID             string `gorm:"primaryKey" json:"id"`
	Username       string `json:"username"`
	HashedPassword []byte `json:"-"`
	Created        int    `gorm:"autoCreateTime" json:"created"`
	Updated        int    `gorm:"autoUpdateTime" json:"updated"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = "usr_" + xid.New().String()
	return nil
}
