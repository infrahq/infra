package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		jerry := &models.Identity{Name: "jseinfeld@infrahq.com"}

		err := CreateIdentity(db, jerry)
		assert.NilError(t, err)

		t.Run("no key id set", func(t *testing.T) {
			key := &models.AccessKey{
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.NilError(t, err)
			assert.Assert(t, key.KeyID != "")
		})

		t.Run("no key secret set", func(t *testing.T) {
			key := &models.AccessKey{
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.NilError(t, err)
			assert.Assert(t, key.Secret != "")
		})

		t.Run("no expiry set", func(t *testing.T) {
			key := &models.AccessKey{
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.NilError(t, err)
			assert.Assert(t, !key.ExpiresAt.IsZero())
		})

		t.Run("no name set", func(t *testing.T) {
			key := &models.AccessKey{
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.NilError(t, err)

			expected := fmt.Sprintf("%s-%s", jerry.Name, key.ID.String())
			assert.Equal(t, key.Name, expected)
		})

		t.Run("invalid specified key id length", func(t *testing.T) {
			key := &models.AccessKey{
				KeyID:      "too-short",
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.Error(t, err, "invalid key length")
		})

		t.Run("invalid specified key secret length", func(t *testing.T) {
			key := &models.AccessKey{
				Secret:     "too-short",
				IssuedFor:  jerry.ID,
				ProviderID: InfraProvider(db).ID,
			}
			_, err := CreateAccessKey(db, key)
			assert.Error(t, err, "invalid secret length")
		})
	})
}

func createAccessKeyWithExtensionDeadline(t *testing.T, db GormTxn, ttl, extensionDeadline time.Duration) (string, *models.AccessKey) {
	identity := &models.Identity{Name: "Wall-E"}
	err := CreateIdentity(db, identity)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:         identity.ID,
		ProviderID:        InfraProvider(db).ID,
		ExpiresAt:         time.Now().Add(ttl),
		ExtensionDeadline: time.Now().Add(extensionDeadline).UTC(),
	}

	body, err := CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, token
}

func TestCheckAccessKeySecret(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		body, _ := createTestAccessKey(t, db, time.Hour*5)

		_, err := ValidateAccessKey(db, body)
		assert.NilError(t, err)

		random := generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric)
		authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

		_, err = ValidateAccessKey(db, authorization)
		assert.Error(t, err, "access key invalid secret")
	})
}

func TestDeleteAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		_, token := createTestAccessKey(t, db, time.Minute*5)

		_, err := GetAccessKey(db, ByID(token.ID))
		assert.NilError(t, err)

		err = DeleteAccessKey(db, token.ID)
		assert.NilError(t, err)

		_, err = GetAccessKey(db, ByID(token.ID))
		assert.Error(t, err, "record not found")

		err = DeleteAccessKeys(db, ByID(token.ID))
		assert.NilError(t, err)
	})
}

func TestCheckAccessKeyExpired(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		body, _ := createTestAccessKey(t, db, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.ErrorIs(t, err, ErrAccessKeyExpired)
	})
}

func TestCheckAccessKeyPastExtensionDeadline(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		body, _ := createAccessKeyWithExtensionDeadline(t, db, 1*time.Hour, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.ErrorIs(t, err, ErrAccessKeyDeadlineExceeded)
	})
}

func TestListAccessKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *DB) {
		user := &models.Identity{Name: "tmp@infrahq.com"}
		err := CreateIdentity(db, user)
		assert.NilError(t, err)

		token := &models.AccessKey{
			Name:       "first",
			Model:      models.Model{ID: 0},
			IssuedFor:  user.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(time.Hour).UTC(),
			KeyID:      "1234567890",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		token = &models.AccessKey{
			Name:       "second",
			Model:      models.Model{ID: 1},
			IssuedFor:  user.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(-time.Hour).UTC(),
			KeyID:      "1234567891",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		token = &models.AccessKey{
			Name:              "third",
			Model:             models.Model{ID: 2},
			IssuedFor:         user.ID,
			ProviderID:        InfraProvider(db).ID,
			ExpiresAt:         time.Now().Add(time.Hour).UTC(),
			ExtensionDeadline: time.Now().Add(-time.Hour).UTC(),
			KeyID:             "1234567892",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		keys, err := ListAccessKeys(db, nil, ByNotExpiredOrExtended())
		assert.NilError(t, err)
		assert.Assert(t, len(keys) == 1)

		keys, err = ListAccessKeys(db, nil)
		assert.NilError(t, err)
		assert.Assert(t, len(keys) == 3)
	})
}

func createTestAccessKey(t *testing.T, db GormTxn, sessionDuration time.Duration) (string, *models.AccessKey) {
	user := &models.Identity{Name: "tmp@infrahq.com"}
	err := CreateIdentity(db, user)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:  user.ID,
		ProviderID: InfraProvider(db).ID,
		ExpiresAt:  time.Now().Add(sessionDuration),
	}

	body, err := CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, token
}
