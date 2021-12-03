package data

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/models"
)

func createToken(t *testing.T, db *gorm.DB, sessionDuration time.Duration) *models.Token {
	user, err := GetUser(db, &models.User{Email: "tmp@infrahq.com"})
	if errors.Is(err, internal.ErrNotFound) {
		createUsers(t, db, models.User{Email: "tmp@infrahq.com"})
		user, err = GetUser(db, &models.User{Email: "tmp@infrahq.com"})
	}

	require.NoError(t, err)

	in := models.Token{
		User:            *user,
		SessionDuration: sessionDuration,
	}

	token, err := CreateToken(db, &in)
	require.NoError(t, err)

	return token
}

func TestCreateToken(t *testing.T) {
	db := setup(t)

	token := createToken(t, db, time.Minute*1)
	require.NotEmpty(t, token.Checksum)
	require.NotEmpty(t, token.Secret)
	require.NotEmpty(t, token.Key)
	require.WithinDuration(t, token.Expires, time.Now(), time.Minute*1)
}

func TestGetToken(t *testing.T) {
	db := setup(t)
	token := createToken(t, db, time.Minute*1)

	fromDB, err := GetToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)
	require.NotEmpty(t, token.Checksum)
	require.Empty(t, fromDB.Secret)
	require.NotEmpty(t, token.Key)
}

func TestGetUserTokenSelector(t *testing.T) {
	db := setup(t)
	token := createToken(t, db, time.Minute*1)

	user, err := GetUser(db, UserTokenSelector(db, token.SessionToken()))
	require.NoError(t, err)
	require.Equal(t, "tmp@infrahq.com", user.Email)
}

func TestCheckTokenExpired(t *testing.T) {
	db := setup(t)
	token := createToken(t, db, time.Minute*1)

	err := CheckTokenExpired(token)
	require.NoError(t, err)

	token = createToken(t, db, time.Minute*-1)

	err = CheckTokenExpired(token)
	require.EqualError(t, err, "token expired")
}

func TestCheckTokenSecret(t *testing.T) {
	db := setup(t)
	token := createToken(t, db, time.Minute*1)

	err := CheckTokenSecret(token, token.SessionToken())
	require.NoError(t, err)

	random := generate.MathRandom(models.TokenSecretLength)
	authorization := fmt.Sprintf("%s%s", token.Key, random)

	err = CheckTokenSecret(token, authorization)
	require.EqualError(t, err, "token invalid secret")
}

func TestDeleteToken(t *testing.T) {
	db := setup(t)
	token := createToken(t, db, time.Minute*1)

	_, err := GetToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)

	err = DeleteToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)

	_, err = GetToken(db, &models.Token{Key: token.Key})
	require.EqualError(t, err, "record not found")

	err = DeleteToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)
}

func createAPIKey(t *testing.T, db *gorm.DB, name string, permissions ...string) *models.APIKey {
	in := models.APIKey{
		Name:        name,
		Permissions: strings.Join(permissions, " "),
	}

	apiKey, err := CreateAPIKey(db, &in)
	require.NoError(t, err)

	return apiKey
}

func TestCreateAPIKey(t *testing.T) {
	db := setup(t)

	apiKey := createAPIKey(t, db, "tmp", "infra.*")
	require.Equal(t, "tmp", apiKey.Name)
	require.Equal(t, "infra.*", apiKey.Permissions)
	require.NotEmpty(t, apiKey.Key)
}

func TestGetAPIKey(t *testing.T) {
	db := setup(t)
	_ = createAPIKey(t, db, "tmp", "infra.*")

	apiKey, err := GetAPIKey(db, &models.APIKey{Name: "tmp"})
	require.NoError(t, err)
	require.Equal(t, "tmp", apiKey.Name)
	require.Equal(t, "infra.*", apiKey.Permissions)
	require.NotEmpty(t, apiKey.Key)
}

func TestListAPIKey(t *testing.T) {
	db := setup(t)
	_ = createAPIKey(t, db, "tmp", "infra.*")
	_ = createAPIKey(t, db, "pmt", "infra.*")
	_ = createAPIKey(t, db, "mtp", "infra.*")

	apiKeys, err := ListAPIKeys(db, &models.APIKey{})
	require.NoError(t, err)
	require.Len(t, apiKeys, 3)

	apiKeys, err = ListAPIKeys(db, &models.APIKey{Name: "tmp"})
	require.NoError(t, err)
	require.Len(t, apiKeys, 1)
}

func TestDeleteAPIKey(t *testing.T) {
	db := setup(t)
	_ = createAPIKey(t, db, "tmp", "infra.*")

	_, err := GetAPIKey(db, &models.APIKey{Name: "tmp"})
	require.NoError(t, err)

	err = DeleteAPIKey(db, &models.APIKey{Name: "tmp"})
	require.NoError(t, err)

	_, err = GetAPIKey(db, &models.APIKey{Name: "tmp"})
	require.EqualError(t, err, "record not found")

	err = DeleteAPIKey(db, &models.APIKey{Name: "tmp"})
	require.NoError(t, err)
}
