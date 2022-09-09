package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestEncryptionKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		t.Run("create", func(t *testing.T) {
			key := &models.EncryptionKey{
				KeyID:     11,
				Name:      "first",
				Encrypted: []byte("encrypted"),
				Algorithm: "better",
				RootKeyID: "main",
			}
			err := CreateEncryptionKey(db, key)
			assert.NilError(t, err)

			expected := &models.EncryptionKey{
				Model: models.Model{
					ID:        uid.ID(999),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				KeyID:     11,
				Name:      "first",
				Encrypted: []byte("encrypted"),
				Algorithm: "better",
				RootKeyID: "main",
			}
			assert.DeepEqual(t, key, expected, cmpModel)
		})

		t.Run("get not found", func(t *testing.T) {
			_, err := GetEncryptionKeyByName(tx, "does-not-exist")
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("get by name", func(t *testing.T) {
			err := CreateEncryptionKey(db, &models.EncryptionKey{
				KeyID:     12,
				Name:      "second",
				Encrypted: []byte("encrypted"),
				Algorithm: "good",
				RootKeyID: "main",
			})
			assert.NilError(t, err)

			actual, err := GetEncryptionKeyByName(tx, "second")
			assert.NilError(t, err)

			expected := &models.EncryptionKey{
				Model: models.Model{
					ID:        uid.ID(999),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				KeyID:     12,
				Name:      "second",
				Encrypted: []byte("encrypted"),
				Algorithm: "good",
				RootKeyID: "main",
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
	})
}
