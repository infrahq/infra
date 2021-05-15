package data

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/boltdb/bolt"
	"github.com/infrahq/infra/internal/util"
)

type User struct {
	ID         string   `json:"id"`
	Email      string   `json:"email"`
	Created    int64    `json:"created"`
	Providers  []string `json:"providers"`
	Permission string   `json:"permission"`
}

func (d *Data) PutUser(u *User) error {
	if u == nil {
		return errors.New("nil user provided")
	}

	if u.ID == "" {
		u.ID = "usr_" + util.RandString(IDLength)
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
