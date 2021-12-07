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

func createUserToken(t *testing.T, db *gorm.DB, sessionDuration time.Duration) *models.Token {
	createUsers(t, db, models.User{Email: "tmp@infrahq.com"})

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

func TestCheckTokenSecret(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	err := CheckTokenSecret(token, token.SessionToken())
	require.NoError(t, err)

	random := generate.MathRandom(models.TokenSecretLength)
	authorization := fmt.Sprintf("%s%s", token.Key, random)

	err = CheckTokenSecret(token, authorization)
	require.EqualError(t, err, "token invalid secret")
}

func TestCreateUserToken(t *testing.T) {
	db := setup(t)

	token := createUserToken(t, db, time.Minute*1)
	require.NotEmpty(t, token.Checksum)
	require.NotEmpty(t, token.Secret)
	require.NotEmpty(t, token.Key)
	require.Len(t, token.SessionToken(), models.TokenLength)
	require.WithinDuration(t, token.Expires, time.Now(), time.Minute*1)
}

func TestGetUserToken(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	fromDB, err := GetToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)
	require.NotEmpty(t, token.Checksum)
	require.Empty(t, fromDB.Secret)
	require.NotEmpty(t, token.Key)
}

func TestGetUserTokenSelector(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	user, err := GetUser(db, UserTokenSelector(db, token.SessionToken()))
	require.NoError(t, err)
	require.Equal(t, "tmp@infrahq.com", user.Email)
}

func TestCheckUserTokenExpired(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	err := CheckTokenExpired(token)
	require.NoError(t, err)

	token = createUserToken(t, db, time.Minute*-1)

	err = CheckTokenExpired(token)
	require.EqualError(t, err, "token expired")
}

func TestDeleteUserToken(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	_, err := GetToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)

	err = DeleteToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)

	_, err = GetToken(db, &models.Token{Key: token.Key})
	require.EqualError(t, err, "record not found")

	err = DeleteToken(db, &models.Token{Key: token.Key})
	require.NoError(t, err)
}

func createAPIToken(t *testing.T, db *gorm.DB, name string, ttl time.Duration, permissions ...string) (*models.APIToken, *models.Token) {
	in := models.APIToken{
		Name:        name,
		Permissions: strings.Join(permissions, " "),
		TTL:         ttl,
	}

	apiToken, tkn, err := CreateAPIToken(db, &in, &models.Token{})
	require.NoError(t, err)

	return apiToken, tkn
}

func TestCreateAPIToken(t *testing.T) {
	db := setup(t)

	apiToken, token := createAPIToken(t, db, "tmp", 1*time.Hour, "infra.*")
	require.Equal(t, "tmp", apiToken.Name)
	require.Equal(t, "infra.*", apiToken.Permissions)
	require.NotEmpty(t, token.Checksum)
	require.NotEmpty(t, token.Secret)
	require.NotEmpty(t, token.Key)
	require.Len(t, token.SessionToken(), models.TokenLength)
	require.WithinDuration(t, token.Expires, time.Now(), 1*time.Hour)
}

func TestGetAPIToken(t *testing.T) {
	db := setup(t)
	ttl := 1 * time.Hour
	_, _ = createAPIToken(t, db, "tmp", ttl, "infra.*")

	apiToken, err := GetAPIToken(db, &models.APIToken{Name: "tmp"})
	require.NoError(t, err)
	require.Equal(t, "tmp", apiToken.Name)
	require.Equal(t, "infra.*", apiToken.Permissions)
	require.Equal(t, ttl.String(), apiToken.TTL.String())
}

func TestListAPIToken(t *testing.T) {
	db := setup(t)
	_, _ = createAPIToken(t, db, "tmp", 1*time.Hour, "infra.*")
	_, _ = createAPIToken(t, db, "pmt", 1*time.Hour, "infra.*")
	_, _ = createAPIToken(t, db, "mtp", 1*time.Hour, "infra.*")

	apiTokens, err := ListAPITokens(db, &models.APIToken{})
	require.NoError(t, err)
	require.Len(t, apiTokens, 3)

	apiTokens, err = ListAPITokens(db, &models.APIToken{Name: "tmp"})
	require.NoError(t, err)
	require.Len(t, apiTokens, 1)
}

func TestDeleteAPIToken(t *testing.T) {
	db := setup(t)
	_, _ = createAPIToken(t, db, "tmp", 1*time.Hour, "infra.*")

	_, err := GetAPIToken(db, &models.APIToken{Name: "tmp"})
	require.NoError(t, err)

	err = DeleteAPIToken(db, &models.APIToken{Name: "tmp"})
	require.NoError(t, err)

	_, err = GetAPIToken(db, &models.APIToken{Name: "tmp"})
	require.EqualError(t, err, "record not found")

	err = DeleteAPIToken(db, &models.APIToken{Name: "tmp"})
	require.NoError(t, err)
}

func TestCheckAPITokenExpired(t *testing.T) {
	db := setup(t)
	_, token := createAPIToken(t, db, "tmp", -1*time.Hour, "infra.*")

	err := CheckTokenExpired(token)
	require.EqualError(t, err, "token expired")
}
