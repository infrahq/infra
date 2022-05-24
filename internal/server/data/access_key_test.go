package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
)

func TestCreateAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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

func createAccessKeyWithExtensionDeadline(t *testing.T, db *gorm.DB, ttl, exensionDeadline time.Duration) (string, *models.AccessKey) {
	identity := &models.Identity{Name: "Wall-E"}
	err := CreateIdentity(db, identity)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:         identity.ID,
		ProviderID:        InfraProvider(db).ID,
		ExpiresAt:         time.Now().Add(ttl),
		ExtensionDeadline: time.Now().Add(exensionDeadline).UTC(),
	}

	body, err := CreateAccessKey(db, token)
	assert.NilError(t, err)

	return body, token
}

func TestCheckAccessKeySecret(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		body, _ := createTestAccessKey(t, db, time.Hour*5)

		_, err := ValidateAccessKey(db, body)
		assert.NilError(t, err)

		random := generate.MathRandom(models.AccessKeySecretLength)
		authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

		_, err = ValidateAccessKey(db, authorization)
		assert.Error(t, err, "access key invalid secret")
	})
}

func TestDeleteAccessKey(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
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
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		body, _ := createTestAccessKey(t, db, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.Error(t, err, "token expired")
	})
}

func TestCheckAccessKeyPastExtensionDeadline(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		body, _ := createAccessKeyWithExtensionDeadline(t, db, 1*time.Hour, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.Error(t, err, "token extension deadline exceeded")
	})
}

func createTestAccessKey(t *testing.T, db *gorm.DB, sessionDuration time.Duration) (string, *models.AccessKey) {
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
