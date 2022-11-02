package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestGetCredentialByUserID(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		cred := &models.Credential{
			IdentityID:      7145,
			PasswordHash:    []byte("password-hash"),
			OneTimePassword: true,
		}

		err := CreateCredential(db, cred)
		assert.NilError(t, err)

		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			actual, err := GetCredentialByUserID(tx, 7145)
			assert.NilError(t, err)

			expected := &models.Credential{
				Model: models.Model{
					ID:        12345,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{
					OrganizationID: db.DefaultOrg.ID,
				},
				IdentityID:      7145,
				PasswordHash:    []byte("password-hash"),
				OneTimePassword: true,
			}
			assert.DeepEqual(t, actual, expected, cmpModel)
		})
		t.Run("not found", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)
			_, err := GetCredentialByUserID(tx, 91234)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
		t.Run("deleted", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			err := DeleteCredential(tx, cred.ID)
			assert.NilError(t, err)

			_, err = GetCredentialByUserID(tx, 7145)
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})
	})
}

func TestDeleteCredential(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		cred := &models.Credential{
			IdentityID:   7145,
			PasswordHash: []byte("password-hash"),
		}

		err := CreateCredential(db, cred)
		assert.NilError(t, err)

		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			err := DeleteCredential(tx, cred.ID)
			assert.NilError(t, err)

			_, err = GetCredentialByUserID(tx, 7145)
			assert.ErrorIs(t, err, internal.ErrNotFound)

			// Delete again to check idempotence
			err = DeleteCredential(tx, cred.ID)
			assert.NilError(t, err)
		})
		t.Run("delete not found", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			err := DeleteCredential(tx, 171717)
			assert.NilError(t, err)
		})
	})
}
