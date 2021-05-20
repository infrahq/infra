package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/infrahq/infra/internal/generate"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

const (
	IDLength        = 12
	SecretKeyLength = 32
)

type User struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Created   int64    `json:"created"`
	Providers []string `json:"providers"`
}

type Token struct {
	ID           string `json:"id"`
	User         string `json:"user"`
	HashedSecret []byte `json:"hashed_secret,omitempty"`
	Created      int64  `json:"created"`
	Expires      int64  `json:"expires"`
}

func NewDB(path string) (db *bolt.DB, err error) {
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return
	}

	db, err = bolt.Open(filepath.Join(path, "infra.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
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

	return db, nil
}

func PutToken(tx *bolt.Tx, token *Token) (sk string, err error) {
	if token.User == "" {
		return "", errors.New("user id cannot be empty")
	}

	id := generate.RandString(IDLength)
	secret := generate.RandString(SecretKeyLength)

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

	b, err := tx.CreateBucketIfNotExists([]byte("tokens"))
	if err != nil {
		return "", err
	}

	if err = b.Put([]byte(token.ID), buf); err != nil {
		return "", err
	}

	sk = "sk_" + id + string(secret)

	return
}

func DeleteToken(tx *bolt.Tx, id string) error {
	if b := tx.Bucket([]byte("tokens")); b != nil {
		return b.Delete([]byte(id))
	}
	return nil
}

func GetToken(tx *bolt.Tx, id string, secret bool) (token *Token, err error) {
	var buf []byte
	b := tx.Bucket([]byte("tokens"))
	if b == nil {
		return nil, errors.New("token bucket does not exist")
	}

	buf = b.Get([]byte(id))
	if buf == nil {
		return nil, errors.New("token does not exist")
	}

	err = json.Unmarshal(buf, &token)
	if err != nil {
		return
	}

	if !secret {
		token.HashedSecret = []byte{}
	}

	return
}

func ListTokens(tx *bolt.Tx, user string) (tokens []Token, err error) {
	tokens = []Token{}

	b := tx.Bucket([]byte("tokens"))
	if b == nil {
		return nil, errors.New("tokens bucket does not exist")
	}

	err = b.ForEach(func(k, v []byte) (err error) {
		var token Token
		if err = json.Unmarshal(v, &token); err != nil {
			return err
		}

		token.HashedSecret = []byte{}

		if user == "" || token.User == user {
			tokens = append(tokens, token)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return tokens, err
}

func PutUser(tx *bolt.Tx, u *User) error {
	if u == nil {
		return errors.New("nil user provided")
	}

	if u.ID == "" {
		u.ID = "usr_" + generate.RandString(IDLength)
	}

	if u.Created == 0 {
		u.Created = time.Now().Unix()
	}

	if u.Providers == nil {
		u.Providers = []string{}
	}

	buf, err := json.Marshal(u)
	if err != nil {
		return err
	}

	b, err := tx.CreateBucketIfNotExists([]byte("users"))
	if err != nil {
		return errors.New("users bucket does not exist")
	}

	existing, err := FindUser(tx, u.Email)
	if err != nil {
		return err
	}

	if existing != nil && existing.ID != u.ID {
		return errors.New("user with different ID already exists with this email")
	}

	return b.Put([]byte(u.ID), buf)
}

func DeleteUser(tx *bolt.Tx, id string) error {
	b := tx.Bucket([]byte("users"))
	if b == nil {
		return nil
	}

	if err := b.Delete([]byte(id)); err != nil {
		return err
	}

	// Delete associated tokens
	b = tx.Bucket([]byte("tokens"))
	if b == nil {
		return errors.New("tokens bucket does not exist")
	}

	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var token Token
		if err := json.Unmarshal(v, &token); err != nil {
			return err
		}
		if token.User == id {
			c.Delete()
		}
	}
	return nil
}

func GetUser(tx *bolt.Tx, id string) (*User, error) {
	var buf []byte

	b := tx.Bucket([]byte("users"))
	if b == nil {
		return nil, errors.New("users bucket does not exist")
	}

	buf = b.Get([]byte(id))
	if buf == nil {
		return nil, errors.New("user does not exist")
	}

	var user User
	if err := json.Unmarshal(buf, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func ListUsers(tx *bolt.Tx) (users []User, err error) {
	users = []User{}

	b := tx.Bucket([]byte("users"))
	if b == nil {
		return nil, errors.New("users bucket does not exist")
	}

	b.ForEach(func(k, v []byte) (err error) {
		var user User
		if err = json.Unmarshal(v, &user); err != nil {
			return err
		}
		users = append(users, user)
		return nil
	})

	return users, err
}

func FindUser(tx *bolt.Tx, email string) (user *User, err error) {
	b := tx.Bucket([]byte("users"))
	if b == nil {
		return nil, errors.New("users bucket does not exist")
	}

	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var found User
		if err := json.Unmarshal(v, &found); err != nil {
			return nil, err
		}
		if found.Email == email {
			return &found, nil
		}
	}
	return user, err
}

func SyncUsers(tx *bolt.Tx, emails []string, provider string) error {
	b := tx.Bucket([]byte("users"))
	if b == nil {
		return errors.New("users bucket does not exist")
	}

	for _, email := range emails {
		user, err := FindUser(tx, email)
		if err != nil {
			return err
		}

		if user == nil {
			user = &User{Email: email}
		}

		providers := user.Providers

		// Add provider to user's provider list
		if len(providers) == 0 {
			user.Providers = []string{provider}
		} else {
			hasProvider := false
			for _, p := range user.Providers {
				if p == provider {
					hasProvider = true
				}
			}
			if !hasProvider {
				user.Providers = append(user.Providers, provider)
				sort.Strings(user.Providers)
			}
		}

		PutUser(tx, user)

		// delete users if no longer in provider
		users, err := ListUsers(tx)
		if err != nil {
			return err
		}

		emailsMap := make(map[string]bool)
		for _, email := range emails {
			emailsMap[email] = true
		}

		for _, user := range users {
			if !emailsMap[user.Email] {
				providers := []string{}
				hasProvider := false

				for _, p := range user.Providers {
					if p == provider {
						hasProvider = true
					} else {
						providers = append(providers, p)
					}
				}

				user.Providers = providers
				if hasProvider && len(providers) == 0 {
					DeleteUser(tx, user.ID)
				} else {
					PutUser(tx, &user)
				}
			}
		}

	}

	return nil
}
