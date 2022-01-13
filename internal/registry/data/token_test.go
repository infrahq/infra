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
		UserID:          user.ID,
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

func TestCreateOrUpdateTokenCreate(t *testing.T) {
	db := setup(t)

	exampleKey := generate.MathRandom(models.TokenKeyLength)

	in := &models.Token{
		SessionDuration: 1 * time.Hour,
		Key:             exampleKey,
	}

	// it should not exist before the call
	_, err := GetToken(db, ByKey(exampleKey))
	require.ErrorIs(t, err, internal.ErrNotFound)

	token, err := CreateOrUpdateToken(db, in, ByKey(exampleKey))
	require.NoError(t, err)
	require.NotEmpty(t, token.Secret)
	require.True(t, time.Now().Before(token.Expires))
	require.Equal(t, in.Key, token.Key)
}

func TestCreateOrUpdateTokenUpdate(t *testing.T) {
	db := setup(t)

	before := &models.Token{
		SessionDuration: 1 * time.Hour,
		Key:             generate.MathRandom(models.TokenKeyLength),
	}

	// it should not exist before the call
	beforeUpdate, err := CreateToken(db, before)
	require.NoError(t, err, internal.ErrNotFound)

	after := &models.Token{
		Key:             before.Key,
		SessionDuration: 2 * time.Hour,
		Secret:          generate.MathRandom(models.TokenSecretLength),
	}

	afterUpdate, err := CreateOrUpdateToken(db, after, ByKey(before.Key))
	require.NoError(t, err)
	require.Equal(t, beforeUpdate.Key, afterUpdate.Key)
	require.Equal(t, beforeUpdate.ID, afterUpdate.ID)
	require.Equal(t, after.Secret, afterUpdate.Secret)
	require.NotEqual(t, beforeUpdate.Checksum, afterUpdate.Checksum)
	require.NotEqual(t, beforeUpdate.Expires, afterUpdate.Expires)
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

	fromDB, err := GetToken(db, ByKey(token.Key))
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

	token := createUserToken(t, db, time.Minute*-1)

	err := CheckTokenExpired(token)
	require.EqualError(t, err, "token expired")
}

func TestRevokeUserToken(t *testing.T) {
	db := setup(t)
	token := createUserToken(t, db, time.Minute*1)

	_, err := GetToken(db, ByKey(token.Key))
	require.NoError(t, err)

	err = DeleteToken(db, ByKey(token.Key))
	require.NoError(t, err)

	_, err = GetToken(db, ByKey(token.Key))
	require.EqualError(t, err, "record not found")

	err = DeleteToken(db, ByKey(token.Key))
	require.NoError(t, err)
}

func createAPIToken(t *testing.T, db *gorm.DB, name string, ttl time.Duration, permissions ...string) (*models.APIToken, *models.Token) {
	apiToken := &models.APIToken{
		Name:        name,
		Permissions: strings.Join(permissions, " "),
		TTL:         ttl,
	}

	tkn, err := CreateAPIToken(db, apiToken)
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

func TestCreateOrUpdateAPITokenCreate(t *testing.T) {
	db := setup(t)

	name := "create-or-update-api-token-create"

	in := &models.APIToken{
		Name:        name,
		Permissions: "infra.users.read",
		TTL:         1 * time.Hour,
	}

	// should not exist before creation
	_, err := GetAPIToken(db, ByName(name))
	require.ErrorIs(t, err, internal.ErrNotFound)

	token := &models.Token{}
	apiToken, err := CreateOrUpdateAPIToken(db, in, token, ByName(name))
	require.NoError(t, err)
	require.Equal(t, in.Name, apiToken.Name)
	require.Equal(t, in.Permissions, apiToken.Permissions)
	require.Equal(t, in.TTL, apiToken.TTL)
	require.NotEmpty(t, token.Secret)
	require.NotEmpty(t, token.Key)
}

func TestCreateOrUpdateAPITokenUpdate(t *testing.T) {
	db := setup(t)

	name := "create-or-update-api-token-update"

	before := &models.APIToken{
		Name:        name,
		Permissions: "infra.users.read",
		TTL:         1 * time.Hour,
	}

	// it should not exist before the call
	_, err := CreateAPIToken(db, before)
	require.NoError(t, err, internal.ErrNotFound)

	after := &models.APIToken{
		Name:        name,
		Permissions: "infra.users.create",
		TTL:         2 * time.Hour,
	}

	afterToken := &models.Token{
		Key:    generate.MathRandom(models.TokenKeyLength),
		Secret: generate.MathRandom(models.TokenLength),
	}

	_, err = CreateOrUpdateAPIToken(db, after, afterToken, ByName(name))
	require.NoError(t, err)
	afterUpdate, err := GetAPIToken(db, ByID(after.ID))
	require.NoError(t, err)
	require.Equal(t, afterUpdate.ID, before.ID)
	require.Equal(t, before.Name, afterUpdate.Name)
	require.Equal(t, after.Permissions, afterUpdate.Permissions)
	require.Equal(t, after.TTL, afterUpdate.TTL)

	// make sure the associated token is updated
	var tokens []models.Token
	err = list(db, &models.Token{}, &tokens, &models.Token{APITokenID: afterUpdate.ID})
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	token := tokens[0]
	require.Equal(t, afterToken.Key, token.Key)
	// need to check the secret, it isn't returned on list
	err = CheckTokenSecret(&token, afterToken.SessionToken())
	require.NoError(t, err)
}

func TestGetAPIToken(t *testing.T) {
	db := setup(t)
	ttl := 1 * time.Hour
	_, _ = createAPIToken(t, db, "tmp", ttl, "infra.*")

	apiToken, err := GetAPIToken(db, ByName("tmp"))
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
	_, tkn := createAPIToken(t, db, "tmp", 1*time.Hour, "infra.*")

	k, err := GetAPIToken(db, ByName("tmp"))
	require.NoError(t, err)

	err = DeleteAPIToken(db, k.ID)
	require.NoError(t, err)

	_, err = GetAPIToken(db, ByName("tmp"))
	require.EqualError(t, err, "record not found")

	_, err = GetToken(db, ByKey(tkn.Key))
	require.EqualError(t, err, "record not found")

	err = DeleteAPIToken(db, k.ID)
	require.Error(t, err, "record not found")
}

func TestCheckAPITokenExpired(t *testing.T) {
	db := setup(t)
	_, token := createAPIToken(t, db, "tmp", -1*time.Hour, "infra.*")

	err := CheckTokenExpired(token)
	require.EqualError(t, err, "token expired")
}
