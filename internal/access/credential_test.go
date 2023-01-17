package access

import (
	"io/fs"
	"os"
	"path"
	"testing"

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

func TestCheckPasswordRequirements(t *testing.T) {
	rCtx := setupAccessTestContext(t)
	t.Run("default password requirements", func(t *testing.T) {
		err := checkPasswordRequirements(rCtx.DBTxn, "password")
		assert.NilError(t, err)

		err = checkPasswordRequirements(rCtx.DBTxn, "passwor")
		assert.DeepEqual(t, err, validate.Error{
			"password": []string{"8 characters"},
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
