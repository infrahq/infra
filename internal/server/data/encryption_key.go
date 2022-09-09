package data

import (
	mathrand "math/rand"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
)

type encryptionKeysTable models.EncryptionKey

func (e encryptionKeysTable) Table() string {
	return "encryption_keys"
}

func (e encryptionKeysTable) Columns() []string {
	return []string{"algorithm", "created_at", "deleted_at", "encrypted", "id", "key_id", "name", "root_key_id", "updated_at"}
}

func (e encryptionKeysTable) Values() []any {
	return []any{e.Algorithm, e.CreatedAt, e.DeletedAt, e.Encrypted, e.ID, e.KeyID, e.Name, e.RootKeyID, e.UpdatedAt}
}

func (e *encryptionKeysTable) ScanFields() []any {
	return []any{&e.Algorithm, &e.CreatedAt, &e.DeletedAt, &e.Encrypted, &e.ID, &e.KeyID, &e.Name, &e.RootKeyID, &e.UpdatedAt}
}

func CreateEncryptionKey(db GormTxn, key *models.EncryptionKey) (*models.EncryptionKey, error) {
	if key.KeyID == 0 {
		// not a security issue; just an identifier
		key.KeyID = mathrand.Int31() // nolint:gosec
	}

	if err := add(db, key); err != nil {
		return nil, err
	}

	return key, nil
}

func GetEncryptionKey(db GormTxn, selector SelectorFunc) (result *models.EncryptionKey, err error) {
	return get[models.EncryptionKey](db, selector)
}

func ByEncryptionKeyID(keyID int32) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key_id = ?", keyID)
	}
}
