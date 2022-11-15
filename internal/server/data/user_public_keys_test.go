package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestUserPublicKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		user := &models.Identity{Name: "main@example.com"}
		other := &models.Identity{Name: "other@example.com"}
		createIdentities(t, tx, user, other)

		publicKey := &models.UserPublicKey{
			UserID:      user.ID,
			PublicKey:   "the-public-key",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint",
		}
		err := AddUserPublicKey(tx, publicKey)
		assert.NilError(t, err)

		second := &models.UserPublicKey{
			UserID:      user.ID,
			PublicKey:   "the-public-key-2",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint-2",
		}
		err = AddUserPublicKey(tx, second)
		assert.NilError(t, err)

		actual, err := userPublicKeys(tx, user.ID)
		assert.NilError(t, err)
		expected := []models.UserPublicKey{*publicKey, *second}
		assert.DeepEqual(t, actual, expected, cmpTimeWithDBPrecision)
	})
}

func TestAddUserPublicKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		user := &models.Identity{Name: "main@example.com"}
		createIdentities(t, tx, user)

		publicKey := &models.UserPublicKey{
			UserID:      user.ID,
			Name:        "the-name",
			PublicKey:   "the-public-key",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint",
		}
		err := AddUserPublicKey(tx, publicKey)
		assert.NilError(t, err)

		expected := &models.UserPublicKey{
			Model: models.Model{
				ID:        999,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			UserID:      user.ID,
			Name:        "the-name",
			PublicKey:   "the-public-key",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint",
		}
		assert.DeepEqual(t, publicKey, expected, cmpModel)
	})
}
