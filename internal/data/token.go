package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/infrahq/infra/internal/util"
	"golang.org/x/crypto/bcrypt"
)

type Token struct {
	ID           string `json:"id"`
	User         string `json:"user"`
	HashedSecret []byte `json:"hashed_secret"`
	Created      int64  `json:"created"`
	Expires      int64  `json:"expires"`
}

func (d *Data) PutToken(token *Token) (sk string, err error) {
	if token.User == "" {
		return "", errors.New("user id cannot be empty")
	}

	id := util.RandString(IDLength)
	secret := util.RandString(SecretKeyLength)

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	token.ID = "tk_" + id
	token.HashedSecret = hashedSecret
	token.Created = time.Now().Unix()
	token.Expires = time.Now().Add(time.Hour * 1).Unix()

	buf, err := json.Marshal(token)
	if err != nil {
		return
	}

	err = d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("tokens"))
		if err != nil {
			return nil
		}

		return b.Put([]byte(token.ID), buf)
	})
	if err != nil {
		return
	}

	sk = "sk_" + id + string(secret)

	return
}

func (d *Data) DeleteToken(id string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("tokens")); b != nil {
			return b.Delete([]byte(id))
		}
		return nil
	})
}

func (d *Data) GetToken(id string, secret bool) (token *Token, err error) {
	var buf []byte
	err = d.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tokens"))
		if b == nil {
			return errors.New("token bucket does not exist")
		}

		buf = b.Get([]byte(id))
		if buf == nil {
			return errors.New("token does not exist")
		}

		return nil
	})
	if err != nil {
		return
	}

	fmt.Println(string(buf))

	err = json.Unmarshal(buf, &token)
	if err != nil {
		return
	}

	if !secret {
		token.HashedSecret = []byte{}
	}

	return
}
