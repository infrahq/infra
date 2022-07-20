package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
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

		random := generate.MathRandom(models.AccessKeySecretLength, generate.CharsetAlphaNumeric)
		authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

		_, err = ValidateAccessKey(db, authorization)
		assert.Error(t, err, "access key invalid secret")
	})
}

func TestDeleteAccessKeys(t *testing.T) {
	setup := func(t *testing.T, db *gorm.DB, user *models.Identity) *models.AccessKey {
		key := &models.AccessKey{
			IssuedFor:  user.ID,
			ProviderID: InfraProvider(db).ID,
			ExpiresAt:  time.Now().Add(10 * time.Second),
		}

		_, err := CreateAccessKey(db, key)
		assert.NilError(t, err)

		_, err = GetAccessKey(db, ByID(key.ID))
		assert.NilError(t, err)

		return key
	}

	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		user := &models.Identity{Name: "tmp@example.com"}
		err := CreateIdentity(db, user)
		assert.NilError(t, err)

		t.Run("delete by ID", func(t *testing.T) {
			key := setup(t, db, user)
			err := DeleteAccessKeys(db, DeleteAccessKeysQuery{ID: key.ID})
			assert.NilError(t, err)

			_, err = GetAccessKey(db, ByID(key.ID))
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("delete by IssuedFor", func(t *testing.T) {
			key := setup(t, db, user)
			err := DeleteAccessKeys(db, DeleteAccessKeysQuery{IssuedFor: key.IssuedFor})
			assert.NilError(t, err)

			_, err = GetAccessKey(db, ByID(key.ID))
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("delete by ProviderID", func(t *testing.T) {
			key := setup(t, db, user)
			err := DeleteAccessKeys(db, DeleteAccessKeysQuery{ProviderID: key.ProviderID})
			assert.NilError(t, err)

			_, err = GetAccessKey(db, ByID(key.ID))
			assert.ErrorIs(t, err, internal.ErrNotFound)
		})

		t.Run("delete missing an ID", func(t *testing.T) {
			err := DeleteAccessKeys(db, DeleteAccessKeysQuery{})
			assert.Error(t, err, "delete requires an ID")
		})

		t.Run("delete non-existent", func(t *testing.T) {
			err := DeleteAccessKeys(db, DeleteAccessKeysQuery{ID: 12345})
			assert.NilError(t, err)
		})
	})
}

func TestValidateAccessKey_Expired(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		body, _ := createTestAccessKey(t, db, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.ErrorIs(t, err, ErrAccessKeyExpired)
	})
}

func TestValidateAccessKey_PastExtensionDeadline(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		body, _ := createAccessKeyWithExtensionDeadline(t, db, 1*time.Hour, -1*time.Hour)

		_, err := ValidateAccessKey(db, body)
		assert.ErrorIs(t, err, ErrAccessKeyDeadlineExceeded)
	})
}

func TestListAccessKeys(t *testing.T) {
	runDBTests(t, func(t *testing.T, db *gorm.DB) {
		user := &models.Identity{Name: "tmp@infrahq.com"}
		err := CreateIdentity(db, user)
		assert.NilError(t, err)

		infraProvider := InfraProvider(db)

		token := &models.AccessKey{
			Name:       "first",
			Model:      models.Model{ID: 0},
			IssuedFor:  user.ID,
			ProviderID: infraProvider.ID,
			ExpiresAt:  time.Now().Add(time.Hour).UTC(),
			KeyID:      "1234567890",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		token = &models.AccessKey{
			Name:       "second",
			Model:      models.Model{ID: 1},
			IssuedFor:  user.ID,
			ProviderID: infraProvider.ID,
			ExpiresAt:  time.Now().Add(-time.Hour).UTC(),
			KeyID:      "1234567891",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		token = &models.AccessKey{
			Name:              "third",
			Model:             models.Model{ID: 2},
			IssuedFor:         user.ID,
			ProviderID:        infraProvider.ID,
			ExpiresAt:         time.Now().Add(time.Hour).UTC(),
			ExtensionDeadline: time.Now().Add(-time.Hour).UTC(),
			KeyID:             "1234567892",
		}
		_, err = CreateAccessKey(db, token)
		assert.NilError(t, err)

		t.Run("defaults", func(t *testing.T) {
			actual, err := ListAccessKeys(db, ListAccessKeysQuery{})
			assert.NilError(t, err)
			expected := []models.AccessKey{
				{Name: "first", IssuedForName: user.Name},
			}
			assert.DeepEqual(t, actual, expected, cmpAccessKeyShallow)
		})

		// TODO: test by issuedFor
		// TODO: test by name
		// TODO: test with expired
		// TODO: test pagination total count is set
		// TODO: test combinations
	})
}

var cmpAccessKeyShallow = gocmp.Comparer(func(x, y models.AccessKey) bool {
	return x.Name == y.Name && x.IssuedForName == y.IssuedForName
})

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
