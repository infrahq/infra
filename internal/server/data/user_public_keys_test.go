package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
)

func TestListUserPublicKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("all", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			user := &models.Identity{Name: "main@example.com"}
			other := &models.Identity{Name: "other@example.com"}
			createIdentities(t, tx, user, other)

			otherKey := &models.UserPublicKey{
				UserID:      other.ID,
				PublicKey:   "the-other-public-key",
				KeyType:     "ssh-rsa",
				Fingerprint: "the-other-fingerprint",
				ExpiresAt:   time.Now().Add(time.Hour),
			}
			err := AddUserPublicKey(tx, otherKey)
			assert.NilError(t, err)

			publicKey := &models.UserPublicKey{
				UserID:      user.ID,
				PublicKey:   "the-public-key",
				KeyType:     "ssh-rsa",
				Fingerprint: "the-fingerprint",
				ExpiresAt:   time.Now().Add(time.Hour),
			}
			err = AddUserPublicKey(tx, publicKey)
			assert.NilError(t, err)

			second := &models.UserPublicKey{
				UserID:      user.ID,
				PublicKey:   "the-public-key-2",
				KeyType:     "ssh-rsa",
				Fingerprint: "the-fingerprint-2",
				ExpiresAt:   time.Now().Add(time.Hour),
			}
			err = AddUserPublicKey(tx, second)
			assert.NilError(t, err)

			expired := &models.UserPublicKey{
				UserID:      user.ID,
				PublicKey:   "the-public-key-3",
				KeyType:     "ssh-rsa",
				Fingerprint: "the-fingerprint-3",
				ExpiresAt:   time.Now().Add(-time.Hour),
			}
			err = AddUserPublicKey(tx, expired)
			assert.NilError(t, err)

			actual, err := listUserPublicKeys(tx, user.ID)
			assert.NilError(t, err)
			expected := []models.UserPublicKey{*publicKey, *second}
			assert.DeepEqual(t, actual, expected, cmpTimeWithDBPrecision)
		})
		t.Run("deleted", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			user := &models.Identity{Name: "main@example.com"}
			createIdentities(t, tx, user)

			publicKey := &models.UserPublicKey{
				UserID:      user.ID,
				PublicKey:   "the-public-key",
				KeyType:     "ssh-rsa",
				Fingerprint: "the-fingerprint",
				ExpiresAt:   time.Now().Add(time.Hour),
			}
			err := AddUserPublicKey(tx, publicKey)
			assert.NilError(t, err)

			assert.NilError(t, DeleteUserPublicKeys(tx, user.ID))

			actual, err := listUserPublicKeys(tx, user.ID)
			assert.NilError(t, err)
			assert.Equal(t, len(actual), 0)
		})
	})
}

func TestAddUserPublicKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		user := &models.Identity{Name: "main@example.com"}
		createIdentities(t, tx, user)

		expiry := time.Now().Add(time.Hour)
		publicKey := &models.UserPublicKey{
			UserID:      user.ID,
			Name:        "the-name",
			PublicKey:   "the-public-key",
			KeyType:     "ssh-rsa",
			Fingerprint: "the-fingerprint",
			ExpiresAt:   expiry,
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
			ExpiresAt:   expiry,
		}

		cmpUserPublicKey := cmp.Options{
			cmpModel,
			cmp.FilterPath(opt.PathField(models.UserPublicKey{}, "ExpiresAt"),
				opt.TimeWithThreshold(2*time.Second)),
		}
		assert.DeepEqual(t, publicKey, expected, cmpUserPublicKey)
	})
}
