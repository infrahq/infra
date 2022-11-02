package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
)

func TestCreatePasswordResetToken(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		tx := txnForTestCase(t, db, db.DefaultOrg.ID)

		token, err := CreatePasswordResetToken(tx, 8222, 5*time.Second)
		assert.NilError(t, err)
		assert.Assert(t, token != "")

		userID, err := ClaimPasswordResetToken(tx, token)
		assert.NilError(t, err)
		assert.Equal(t, userID, uid.ID(8222))
	})
}

func TestClaimPasswordResetToken(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		t.Run("deletes token", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			token, err := CreatePasswordResetToken(tx, 8222, 5*time.Second)
			assert.NilError(t, err)
			assert.Assert(t, token != "")

			userID, err := ClaimPasswordResetToken(tx, token)
			assert.NilError(t, err)
			assert.Equal(t, userID, uid.ID(8222))

			// Get again should fail because it was deleted
			_, err = ClaimPasswordResetToken(tx, token)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("expired token", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			token, err := CreatePasswordResetToken(tx, 8222, -5*time.Second)
			assert.NilError(t, err)
			assert.Assert(t, token != "")

			_, err = ClaimPasswordResetToken(tx, token)
			assert.ErrorIs(t, err, internal.ErrExpired)
		})
	})
}

func TestRemoveExpiredPasswordResetTokens(t *testing.T) {
	tx := setupDB(t)
	token, err := CreatePasswordResetToken(tx, uid.New(), -1)
	assert.NilError(t, err)

	err = RemoveExpiredPasswordResetTokens(tx)
	assert.NilError(t, err)

	row := tx.QueryRow("select count(*) from password_reset_tokens where token = ?", token)
	assert.NilError(t, row.Err())
	var count int64
	err = row.Scan(&count)
	assert.NilError(t, err)
	assert.Assert(t, count == 0)
}
