package access

import (
	"testing"
	"unicode"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestSettingsPasswordRequirements(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	_, err = CreateCredential(c, *user)
	assert.NilError(t, err)

	err = data.UpdateSettings(db, &models.Settings{
		LengthMin: 8,
	})
	assert.NilError(t, err)
	t.Run("Update user credentials fails if less than min length", func(t *testing.T) {
		err := UpdateCredential(c, user, "", "short")
		assert.ErrorContains(t, err, "validation failed: password")
		assert.ErrorContains(t, err, "needs minimum length of 8")
	})

	// Test min length success
	settings, err := data.GetSettings(db)
	assert.NilError(t, err)
	settings.LengthMin = 5
	err = data.UpdateSettings(db, settings)
	assert.NilError(t, err)
	t.Run("Update user credentials passes if equal than min length", func(t *testing.T) {
		err := UpdateCredential(c, user, "", "short")
		assert.NilError(t, err)
	})
	t.Run("Update user credentials passes if equal than min length", func(t *testing.T) {
		err := UpdateCredential(c, user, "", "longer")
		assert.NilError(t, err)
	})

	// Test multiple failures
	settings.LengthMin = 10
	settings.SymbolMin = 1
	err = data.UpdateSettings(db, settings)
	assert.NilError(t, err)
	t.Run("Update user credentials fails with multiple requirement failures", func(t *testing.T) {
		err := UpdateCredential(c, user, "", "badpw")
		assert.ErrorContains(t, err, "validation failed: password:")
		assert.ErrorContains(t, err, "needs minimum 1 symbols")
		assert.ErrorContains(t, err, "needs minimum length of 10")
	})
}

func TestCreateCredential(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	oneTimePassword, err := CreateCredential(c, *user)
	assert.NilError(t, err)
	assert.Assert(t, oneTimePassword != "")

	_, err = data.GetCredentialByUserID(db, user.ID)
	assert.NilError(t, err)
}

func TestUpdateCredentials(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}

	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	tmpPassword, err := CreateCredential(c, *user)
	assert.NilError(t, err)

	userCreds, err := data.GetCredentialByUserID(db, user.ID)
	assert.NilError(t, err)

	t.Run("Update user credentials IS single use password", func(t *testing.T) {
		err := UpdateCredential(c, user, "", "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredentialByUserID(db, user.ID)
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, true)
	})

	t.Run("Update own credentials is NOT single use password", func(t *testing.T) {
		err := data.UpdateCredential(db, userCreds)
		assert.NilError(t, err)

		rCtx := GetRequestContext(c)
		rCtx.Authenticated.User = user
		c.Set(RequestContextKey, rCtx)

		err = UpdateCredential(c, user, tmpPassword, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredentialByUserID(db, user.ID)
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, false)
	})

	t.Run("Update own credentials removes password reset scope, but keeps other scopes", func(t *testing.T) {
		err := data.UpdateCredential(db, userCreds)
		assert.NilError(t, err)

		rCtx := GetRequestContext(c)
		rCtx.Authenticated.User = user

		key := &models.AccessKey{
			IssuedFor:  user.ID,
			ProviderID: data.InfraProvider(db).ID,
			Scopes: []string{
				models.ScopeAllowCreateAccessKey,
				models.ScopePasswordReset,
			},
		}
		_, err = CreateAccessKey(c, key)
		assert.NilError(t, err)
		rCtx.Authenticated.AccessKey = key
		c.Set(RequestContextKey, rCtx)

		err = UpdateCredential(c, user, "", "newPassword")
		assert.ErrorContains(t, err, "oldPassword: is required")

		err = UpdateCredential(c, user, "somePassword", "newPassword")
		assert.ErrorContains(t, err, "oldPassword: invalid oldPassword")

		err = UpdateCredential(c, user, tmpPassword, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredentialByUserID(db, user.ID)
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, false)

		updatedKey, err := data.GetAccessKeyByKeyID(db, key.KeyID)
		assert.NilError(t, err)
		assert.DeepEqual(t, updatedKey.Scopes, models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey})
	})
}

func TestHasMinimumCount(t *testing.T) {
	assert.Assert(t, !hasMinimumCount(2, "aB1!", unicode.IsLower))
	assert.Assert(t, !hasMinimumCount(2, "aB1!", unicode.IsUpper))
	assert.Assert(t, !hasMinimumCount(2, "aB1!", unicode.IsNumber))
	assert.Assert(t, !hasMinimumCount(2, "aB1!", isSymbol))

	assert.Assert(t, hasMinimumCount(2, "aaBB11!!", unicode.IsLower))
	assert.Assert(t, hasMinimumCount(2, "aaBB11!!", unicode.IsUpper))
	assert.Assert(t, hasMinimumCount(2, "aaBB11!!", unicode.IsNumber))
	assert.Assert(t, hasMinimumCount(2, "aaBB11!!", isSymbol))
}

func TestIsSymbol(t *testing.T) {
	symbols := " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	for _, s := range symbols {
		assert.Assert(t, isSymbol(s))
	}

	notSymbols := "abcABC123"
	for _, ns := range notSymbols {
		assert.Assert(t, !isSymbol(ns))
	}
}
