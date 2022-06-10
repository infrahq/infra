package access

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
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

	creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	assert.NilError(t, err)

	assert.Equal(t, creds.OneTimePasswordUsed, false)
}

func TestUpdateCredentials(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	username := "bruce@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	_, err = CreateCredential(c, *user)
	assert.NilError(t, err)

	t.Run("Update user credentials IS single use password", func(t *testing.T) {
		err := UpdateCredential(c, user, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, true)
	})

	t.Run("Update own credentials is NOT single use password", func(t *testing.T) {
		c.Set("identity", user)

		err := UpdateCredential(c, user, "newPassword")
		assert.NilError(t, err)

		creds, err := data.GetCredential(db, data.ByIdentityID(user.ID))
		assert.NilError(t, err)
		assert.Equal(t, creds.OneTimePassword, false)
	})
}
