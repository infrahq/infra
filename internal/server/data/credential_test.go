package data

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateCredential(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		cred := &models.Credential{
			IdentityID:      7145,
			PasswordHash:    []byte("password-hash"),
			OneTimePassword: true,
		}

		err := CreateCredential(db, cred)
		assert.NilError(t, err)

		created, err := GetCredentialByUserID(db, cred.IdentityID)
		assert.NilError(t, err)

		expected := &models.Credential{
			Model: models.Model{
				ID:        cred.ID,
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
		assert.DeepEqual(t, expected, created, cmpModel)
	})
}

func TestUpdateCredential(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		past := time.Date(2022, 1, 2, 3, 4, 5, 600, time.UTC)
		cred := &models.Credential{
			Model: models.Model{
				CreatedAt: past,
				UpdatedAt: past,
			},
			IdentityID:      7145,
			PasswordHash:    []byte("password-hash"),
			OneTimePassword: true,
		}

		err := CreateCredential(db, cred)
		assert.NilError(t, err)

		t.Run("success", func(t *testing.T) {
			tx := txnForTestCase(t, db, db.DefaultOrg.ID)

			updated := *cred // shallow copy
			updated.PasswordHash = []byte("new-hash")
			updated.OneTimePassword = false

			err := UpdateCredential(tx, &updated)
			assert.NilError(t, err)

			actual, err := GetCredentialByUserID(tx, cred.IdentityID)
			assert.NilError(t, err)

			expected := &models.Credential{
				Model: models.Model{
					ID:        cred.ID,
					CreatedAt: past,
					UpdatedAt: time.Now(),
				},
				OrganizationMember: models.OrganizationMember{
					OrganizationID: db.DefaultOrg.ID,
				},
				IdentityID:   7145,
				PasswordHash: []byte("new-hash"),
			}
			assert.DeepEqual(t, expected, actual, cmpModel)
		})
	})
}

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
