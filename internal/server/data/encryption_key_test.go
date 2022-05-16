package data

import (
	"testing"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestEncryptionKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		k, err := CreateEncryptionKey(db, &models.EncryptionKey{
			Name:      "foo",
			Encrypted: []byte{0x00},
			Algorithm: "foo",
		})
		assert.NilError(t, err)

		assert.Assert(t, k.KeyID != 0)

		k2, err := GetEncryptionKey(db, ByEncryptionKeyID(k.KeyID))
		assert.NilError(t, err)

		assert.Equal(t, "foo", k2.Name)

		k3, err := GetEncryptionKey(db, ByName("foo"))
		assert.NilError(t, err)

		assert.Equal(t, k.ID, k3.ID)
	})
}
