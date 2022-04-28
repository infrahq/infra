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

func createAccessKey(t *testing.T, db *gorm.DB, sessionDuration time.Duration) (string, *models.AccessKey) {
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
	db := setup(t)
	body, _ := createAccessKey(t, db, time.Hour*5)

	_, err := ValidateAccessKey(db, body)
	assert.NilError(t, err)

	random := generate.MathRandom(models.AccessKeySecretLength)
	authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

	_, err = ValidateAccessKey(db, authorization)
	assert.Error(t, err, "access key invalid secret")
}

func TestDeleteAccessKey(t *testing.T) {
	db := setup(t)
	_, token := createAccessKey(t, db, time.Minute*5)

	_, err := GetAccessKey(db, ByID(token.ID))
	assert.NilError(t, err)

	err = DeleteAccessKey(db, token.ID)
	assert.NilError(t, err)

	_, err = GetAccessKey(db, ByID(token.ID))
	assert.Error(t, err, "record not found")

	err = DeleteAccessKeys(db, ByID(token.ID))
	assert.NilError(t, err)
}

func TestCheckAccessKeyExpired(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKey(t, db, -1*time.Hour)

	_, err := ValidateAccessKey(db, body)
	assert.Error(t, err, "token expired")
}

func TestCheckAccessKeyPastExtensionDeadline(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKeyWithExtensionDeadline(t, db, 1*time.Hour, -1*time.Hour)

	_, err := ValidateAccessKey(db, body)
	assert.Error(t, err, "token extension deadline exceeded")
}
