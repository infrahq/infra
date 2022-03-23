package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
)

func createAccessKey(t *testing.T, db *gorm.DB, sessionDuration time.Duration) (string, *models.AccessKey) {
	user := &models.User{Email: "tmp@infrahq.com"}
	err := CreateUser(db, user)
	require.NoError(t, err)

	token := &models.AccessKey{
		IssuedFor: user.PolyID(),
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	body, err := CreateAccessKey(db, token)
	require.NoError(t, err)

	return body, token
}

func createAccessKeyWithExtensionDeadline(t *testing.T, db *gorm.DB, ttl, exensionDeadline time.Duration) (string, *models.AccessKey) {
	machine := &models.Machine{Name: "Wall-E"}
	err := CreateMachine(db, machine)
	require.NoError(t, err)

	token := &models.AccessKey{
		IssuedFor:         machine.PolyID(),
		ExpiresAt:         time.Now().Add(ttl),
		ExtensionDeadline: time.Now().Add(exensionDeadline),
	}

	body, err := CreateAccessKey(db, token)
	require.NoError(t, err)

	return body, token
}

func TestCheckAccessKeySecret(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKey(t, db, time.Hour*5)

	_, err := ValidateAccessKey(db, body)
	require.NoError(t, err)

	random := generate.MathRandom(models.AccessKeySecretLength)
	authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

	_, err = ValidateAccessKey(db, authorization)
	require.EqualError(t, err, "access key invalid secret")
}

func TestDeleteAccessKey(t *testing.T) {
	db := setup(t)
	_, token := createAccessKey(t, db, time.Minute*5)

	_, err := GetAccessKey(db, ByID(token.ID))
	require.NoError(t, err)

	err = DeleteAccessKey(db, token.ID)
	require.NoError(t, err)

	_, err = GetAccessKey(db, ByID(token.ID))
	require.EqualError(t, err, "record not found")

	err = DeleteAccessKeys(db, ByID(token.ID))
	require.NoError(t, err)
}

func TestCheckAccessKeyExpired(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKey(t, db, -1*time.Hour)

	_, err := ValidateAccessKey(db, body)
	require.EqualError(t, err, "token expired")
}

func TestCheckAccessKeyPastExtensionDeadline(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKeyWithExtensionDeadline(t, db, 1*time.Hour, -1*time.Hour)

	_, err := ValidateAccessKey(db, body)
	require.EqualError(t, err, "token extension deadline exceeded")
}
