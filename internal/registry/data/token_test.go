package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createAPIToken(t *testing.T, db *gorm.DB, sessionDuration time.Duration) (string, *models.APIToken) {
	user := &models.User{Email: "tmp@infrahq.com"}
	err := CreateUser(db, user)
	require.NoError(t, err)

	token := &models.APIToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	body, err := CreateAPIToken(db, token)
	require.NoError(t, err)

	return body, token
}

func TestCheckAPITokenSecret(t *testing.T) {
	db := setup(t)
	body, _ := createAPIToken(t, db, time.Hour*5)

	_, err := LookupAPIToken(db, body)
	require.NoError(t, err)

	random := generate.MathRandom(models.APITokenSecretLength)
	authorization := fmt.Sprintf("%s.%s", strings.Split(body, ".")[0], random)

	_, err = LookupAPIToken(db, authorization)
	require.EqualError(t, err, "token invalid secret")
}

func TestDeleteAPIToken(t *testing.T) {
	db := setup(t)
	_, token := createAPIToken(t, db, time.Minute*5)

	_, err := GetAPIToken(db, ByID(token.ID))
	require.NoError(t, err)

	err = DeleteAPIToken(db, token.ID)
	require.NoError(t, err)

	_, err = GetAPIToken(db, ByID(token.ID))
	require.EqualError(t, err, "record not found")

	err = DeleteAPITokens(db, ByID(token.ID))
	require.NoError(t, err)
}

func TestCheckAPITokenExpired(t *testing.T) {
	db := setup(t)
	body, _ := createAPIToken(t, db, -1*time.Hour)

	_, err := LookupAPIToken(db, body)
	require.EqualError(t, err, "token expired")
}
