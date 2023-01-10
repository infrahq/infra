package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
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

func TestDeleteExpiredUserPublicKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)
		user := &models.Identity{Name: "user@example.com"}
		createIdentities(t, tx, user)

		uk := &models.UserPublicKey{
			Name:        "expired",
			UserID:      user.ID,
			Fingerprint: "fingerprint-1",
			PublicKey:   "key",
			KeyType:     "ssh-rsa",
			ExpiresAt:   time.Now().Add(-2 * time.Hour),
		}
		uk2 := &models.UserPublicKey{
			Name:        "not expired",
			UserID:      user.ID,
			Fingerprint: "fingerprint-2",
			PublicKey:   "key",
			KeyType:     "ssh-rsa",
			ExpiresAt:   time.Now().Add(10 * time.Minute),
		}
		uk3 := &models.UserPublicKey{
			Name:        "already deleted",
			UserID:      user.ID,
			Fingerprint: "fingerprint-3",
			PublicKey:   "key",
			KeyType:     "ssh-rsa",
			ExpiresAt:   time.Now().Add(-2 * time.Hour),
		}
		uk3.DeletedAt.Valid = true
		uk3.DeletedAt.Time = time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC)
		createUserPublicKeys(t, tx, uk, uk2, uk3)

		err := DeleteExpiredUserPublicKeys(tx)
		assert.NilError(t, err)

		remaining, err := listUserPublicKeys(tx, user.ID)
		assert.NilError(t, err)
		expected := []models.UserPublicKey{*uk2}
		assert.DeepEqual(t, remaining, expected, cmpTimeWithDBPrecision)

		deletedAt := getUserPublicKeyDeletedAtByID(t, tx, uk3.ID)
		assert.DeepEqual(t, deletedAt, uk3.DeletedAt.Time, cmpTimeWithDBPrecision)
	})
}

func createUserPublicKeys(t *testing.T, tx WriteTxn, keys ...*models.UserPublicKey) {
	t.Helper()
	for i, k := range keys {
		err := AddUserPublicKey(tx, k)
		assert.NilError(t, err, "public key %d", i)
	}
}

func getUserPublicKeyDeletedAtByID(t *testing.T, tx ReadTxn, id uid.ID) time.Time {
	t.Helper()
	var deletedAt time.Time
	stmt := `SELECT deleted_at FROM user_public_keys WHERE id = ?`
	err := tx.QueryRow(stmt, id).Scan(&deletedAt)
	assert.NilError(t, err)
	return deletedAt
}
