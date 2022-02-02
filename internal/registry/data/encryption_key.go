package data

import (
	mathrand "math/rand"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

func CreateEncryptionKey(db *gorm.DB, key *models.EncryptionKey) (*models.EncryptionKey, error) {
	if key.KeyID == 0 {
		// not a security issue; just an identifier
		key.KeyID = mathrand.Int31() // nolint:gosec
	}

	if err := add(db, key); err != nil {
		return nil, err
	}

	return key, nil
}

func GetEncryptionKey(db *gorm.DB, selector SelectorFunc) (result *models.EncryptionKey, err error) {
	return get[models.EncryptionKey](db, selector)
}

func ByEncryptionKeyID(keyID int32) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key_id = ?", keyID)
	}
}
