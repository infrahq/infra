package access

import (
	"io/fs"
	"os"
	"path"
	"testing"
	"unicode"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
)

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
	assert.Assert(t, !hasMinimumCount("aB1!", 2, unicode.IsLower))
	assert.Assert(t, !hasMinimumCount("aB1!", 2, unicode.IsUpper))
	assert.Assert(t, !hasMinimumCount("aB1!", 2, unicode.IsNumber))
	assert.Assert(t, !hasMinimumCount("aB1!", 2, isSymbol))

	assert.Assert(t, hasMinimumCount("aaBB11!!", 2, unicode.IsLower))
	assert.Assert(t, hasMinimumCount("aaBB11!!", 2, unicode.IsUpper))
	assert.Assert(t, hasMinimumCount("aaBB11!!", 2, unicode.IsNumber))
	assert.Assert(t, hasMinimumCount("aaBB11!!", 2, isSymbol))
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

func TestCheckPasswordRequirements(t *testing.T) {
	_, db, _ := setupAccessTestContext(t)

	t.Run("default password requirements", func(t *testing.T) {
		err := checkPasswordRequirements(db, "password")
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "passwor")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters"},
		})
	})

	t.Run("uppercase letter", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.UppercaseMin = 1
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "1 uppercase letter"},
		})
	})

	t.Run("number", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.NumberMin = 1
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "1 uppercase letter", "1 number"},
		})
	})

	t.Run("symbol", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.SymbolMin = 1
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "1 uppercase letter", "1 number", "1 symbol"},
		})
	})

	t.Run("more than 1 uppercase letter", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.UppercaseMin = 2
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "2 uppercase letters", "1 number", "1 symbol"},
		})
	})

	t.Run("more than 1 number", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.NumberMin = 2
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "2 uppercase letters", "2 numbers", "1 symbol"},
		})
	})

	t.Run("more than 1 symbol", func(t *testing.T) {
		settings, err := data.GetSettings(db)
		assert.NilError(t, err)

		settings.SymbolMin = 2
		err = data.UpdateSettings(db, settings)
		assert.NilError(t, err)

		err = checkPasswordRequirements(db, "password")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters", "2 uppercase letters", "2 numbers", "2 symbols"},
		})
	})
}

func TestCheckBadPasswords(t *testing.T) {
	t.Run("env var not set", func(t *testing.T) {
		err := checkBadPasswords("badpassword")
		assert.NilError(t, err)
	})

	badPasswordsFile := path.Join(t.TempDir(), "bad-passwords")
	t.Setenv("INFRA_SERVER_BAD_PASSWORDS_FILE", badPasswordsFile)
	t.Cleanup(func() {
		os.Remove(badPasswordsFile)
	})

	t.Run("bad passwords file not found", func(t *testing.T) {
		err := checkBadPasswords("badpassword")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	badPasswords := "badpassword"

	err := os.WriteFile(badPasswordsFile, []byte(badPasswords), 0o600)
	assert.NilError(t, err)

	t.Run("password in bad passwords file", func(t *testing.T) {
		err := checkBadPasswords("badpassword")
		assert.ErrorContains(t, err, "cannot use a common password")
	})

	t.Run("password not in bad passwords file", func(t *testing.T) {
		err := checkBadPasswords("notbadpassword")
		assert.NilError(t, err)
	})
}
