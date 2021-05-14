package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	bolt "github.com/boltdb/bolt"
	"golang.org/x/crypto/bcrypt"
)

const (
	IDLength        = 12
	SecretKeyLength = 32
)

var Permissions = []string{"view", "edit", "admin"}

type Data struct {
	db *bolt.DB
}

type User struct {
	ID         string   `json:"id"`
	Email      string   `json:"email"`
	Created    int64    `json:"created"`
	Providers  []string `json:"providers"`
	Permission string   `json:"permission"`
}

type Token struct {
	ID           string `json:"id"`
	User         string `json:"user"`
	HashedSecret []byte `json:"hashed_secret"`
	Created      int64  `json:"created"`
	Expires      int64  `json:"expires"`
}

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

func NewData(dbpath string) (data *Data, err error) {
	if err = os.MkdirAll(dbpath, os.ModePerm); err != nil {
		return
	}

	db, err := bolt.Open(filepath.Join(dbpath, "infra.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return
	}

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return nil
		}

		_, err = tx.CreateBucketIfNotExists([]byte("tokens"))
		if err != nil {
			return nil
		}

		return nil
	})

	return &Data{db: db}, nil
}

func (d *Data) Close() error {
	return d.db.Close()
}

func (d *Data) PutUser(u *User) error {
	if u == nil {
		return errors.New("nil user provided")
	}

	if u.ID == "" {
		u.ID = "usr_" + randString(IDLength)
	}

	if u.Created == 0 {
		u.Created = time.Now().Unix()
	}

	if u.Permission == "" {
		u.Permission = "view"
	}

	if u.Providers == nil {
		u.Providers = []string{}
	}

	buf, err := json.Marshal(u)
	if err != nil {
		return err
	}

	err = d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return errors.New("users bucket does not exist")
		}

		return b.Put([]byte(u.ID), buf)
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *Data) FindUser(email string) (user *User, err error) {
	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var cur User
			if err := json.Unmarshal(v, &cur); err != nil {
				return err
			}
			if cur.Email == email {
				user = &cur
				return nil
			}
		}
		return nil
	})

	return user, err
}

func (d *Data) DeleteUser(id string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("users")); b != nil {
			return b.Delete([]byte(id))
		}
		return nil
	})
}

func (d *Data) GetUser(id string) (*User, error) {
	var buf []byte

	d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return errors.New("users bucket does not exist")
		}

		buf = b.Get([]byte(id))
		if buf == nil {
			return errors.New("user does not exist")
		}

		return nil
	})

	var user User
	if err := json.Unmarshal(buf, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (d *Data) ListUsers() (users []User, err error) {
	users = []User{}

	err = d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return errors.New("users bucket does not exist")
		}

		return b.ForEach(func(k, v []byte) (err error) {
			var user User
			if err = json.Unmarshal(v, &user); err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
	})

	sort.Slice(users, func(i, j int) bool {
		return users[i].Created > users[j].Created
	})

	return users, err
}

// TODO(jmorganca): make this PutToken and change params to accept a *Token
func (d *Data) CreateToken(user string) (token *Token, sk string, err error) {
	id := randString(IDLength)
	secret := randString(SecretKeyLength)

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	token = &Token{
		ID:           "tk_" + id,
		User:         user,
		HashedSecret: hashedSecret,
		Created:      time.Now().Unix(),
		Expires:      time.Now().Add(time.Hour * 1).Unix(),
	}

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

	token.HashedSecret = []byte{}

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
