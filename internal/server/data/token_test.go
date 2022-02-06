package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createAccessKey(t *testing.T, db *gorm.DB, sessionDuration time.Duration) (string, *models.AccessKey) {
	user := &models.User{Email: "tmp@infrahq.com"}
	err := CreateUser(db, user)
	require.NoError(t, err)

	token := &models.AccessKey{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	body, err := CreateAccessKey(db, token)
	require.NoError(t, err)

	return body, token
}

func TestCheckAccessKeySecret(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKey(t, db, time.Hour*5)

	_, err := LookupAccessKey(db, body)
	require.NoError(t, err)

	random := generate.MathRandom(models.AccessKeySecretLength)
	authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

	_, err = LookupAccessKey(db, authorization)
	require.EqualError(t, err, "access key invalid secret")
}

func TestDeleteAccessKey(t *testing.T) {
	db := setup(t)
	_, token := createAccessKey(t, db, time.Minute*5)

	_, err := GetAccessKeys(db, ByID(token.ID))
	require.NoError(t, err)

	err = DeleteAccessKey(db, token.ID)
	require.NoError(t, err)

	_, err = GetAccessKeys(db, ByID(token.ID))
	require.EqualError(t, err, "record not found")

	err = DeleteAccessKeys(db, ByID(token.ID))
	require.NoError(t, err)
}

func TestCheckAccessKeyExpired(t *testing.T) {
	db := setup(t)
	body, _ := createAccessKey(t, db, -1*time.Hour)

	_, err := LookupAccessKey(db, body)
	require.EqualError(t, err, "token expired")
}
