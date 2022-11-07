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
