package main

import (
	"crypto/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	TOKEN_LENGTH = 48
)

func randString(n int) string {
	if n < 0 {
		return randString(0)
	}

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

const (
	USER_ID_PREFIX         = "usr"
	TOKEN_ID_PREFIX        = "tk"
	TOKEN_SECRET_ID_PREFIX = "sk"
)

type User struct {
	ID      string `gorm:"primaryKey" json:"id"`
	Email   string `gorm:"unique" json:"email"`
	Created int    `gorm:"autoCreateTime" json:"created"`
	Updated int    `gorm:"autoUpdateTime" json:"updated"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = USER_ID_PREFIX + "_" + randString(16)
	return nil
}

type Token struct {
	ID           string `gorm:"primaryKey" json:"id"`
	HashedSecret []byte `json:"-"`
	Created      int    `gorm:"autoCreateTime" json:"created"`
	Updated      int    `gorm:"autoUpdateTime" json:"updated"`
	Expires      int    `gorm:"autoUpdateTime" json:"expires"`
	UserID       string
}

func NewToken(db *gorm.DB, userID string) (created *Token, token string, err error) {
	id := randString(16)
	secret := randString(32)

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	created = &Token{
		UserID:       userID,
		ID:           TOKEN_ID_PREFIX + "_" + id,
		HashedSecret: hashedSecret,
		Expires:      int(time.Now().Add(time.Hour * 1).Unix()),
	}

	db.Create(created)

	token = TOKEN_SECRET_ID_PREFIX + "_" + id + secret
	return
}
