package data

import (
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
)

const (
	IDLength        = 12
	SecretKeyLength = 32
)

type Data struct {
	db *bolt.DB
}

func NewData(path string) (data *Data, err error) {
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return
	}

	db, err := bolt.Open(filepath.Join(path, "infra.db"), 0600, &bolt.Options{Timeout: 1 * time.Second})
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
