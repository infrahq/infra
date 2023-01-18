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
	rCtx := setupAccessTestContext(t)
	db := rCtx.DBTxn

	user := &models.Identity{Name: "bruce@example.com"}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	password, err := CreateCredential(rCtx, user)
	assert.NilError(t, err)
	assert.Assert(t, password != "")

	credential, err := data.GetCredentialByUserID(db, user.ID)
	assert.NilError(t, err)
	assert.Assert(t, credential.OneTimePassword)

	_, err = data.GetProviderUser(db, data.InfraProvider(db).ID, user.ID)
	assert.NilError(t, err)
}

func TestUpdateCredentials(t *testing.T) {
	rCtx := setupAccessTestContext(t)

	user := &models.Identity{Name: "bruce@example.com"}

	err := data.CreateIdentity(rCtx.DBTxn, user)
	assert.NilError(t, err)

	oldPassword, err := CreateCredential(rCtx, user)
	assert.NilError(t, err)

	rCtx.Authenticated.User = user

	err = UpdateCredential(rCtx, user, oldPassword, "supersecret")
	assert.NilError(t, err)

	credential, err := data.GetCredentialByUserID(rCtx.DBTxn, user.ID)
	assert.NilError(t, err)
	assert.Assert(t, !credential.OneTimePassword)
}

func TestResetCredentials(t *testing.T) {
	rCtx := setupAccessTestContext(t)
	db := rCtx.DBTxn

	user := &models.Identity{Name: "bruce@example.com"}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	oldPassword, err := CreateCredential(rCtx, user)
	assert.NilError(t, err)
	assert.Assert(t, oldPassword != "")

	credential, err := data.GetCredentialByUserID(db, user.ID)
	assert.NilError(t, err)
	assert.Assert(t, credential.OneTimePassword)

	err = UpdateCredential(rCtx, user, oldPassword, "supersecret")
	assert.NilError(t, err)

	credential, err = data.GetCredentialByUserID(db, user.ID)
	assert.NilError(t, err)
	assert.Assert(t, !credential.OneTimePassword)

	t.Run("reset to random value", func(t *testing.T) {
		newPassword, err := ResetCredential(rCtx, user, "")
		assert.NilError(t, err)
		assert.Assert(t, newPassword != "supersecret")
		assert.Assert(t, newPassword != oldPassword)

		credential, err = data.GetCredentialByUserID(db, user.ID)
		assert.NilError(t, err)
		assert.Assert(t, credential.OneTimePassword)
	})

	t.Run("reset to passed in value", func(t *testing.T) {
		newPassword, err := ResetCredential(rCtx, user, "mypassword")
		assert.NilError(t, err)
		assert.Equal(t, newPassword, "mypassword")

		credential, err = data.GetCredentialByUserID(db, user.ID)
		assert.NilError(t, err)
		assert.Assert(t, credential.OneTimePassword)
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
	rCtx := setupAccessTestContext(t)
	db := rCtx.DBTxn

	t.Run("default password requirements", func(t *testing.T) {
		err := checkPasswordRequirements(rCtx.DBTxn, "password")
		assert.NilError(t, err)

		err = checkPasswordRequirements(rCtx.DBTxn, "passwor")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters"},
		})
	})

	t.Run("uppercase letter", func(t *testing.T) {
		settings, err := data.GetSettings(rCtx.DBTxn)
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
