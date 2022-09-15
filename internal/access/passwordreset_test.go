package access

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestPasswordResetFlow(t *testing.T) {
	c, db, _ := setupAccessTestContext(t)

	user := &models.Identity{Name: "joe@example.com"}

	// setup user
	err := CreateIdentity(c, user)
	assert.NilError(t, err)

	err = data.CreateCredential(db, &models.Credential{
		IdentityID:   user.ID,
		PasswordHash: []byte("foo"),
	})
	assert.NilError(t, err)

	// request password reset
	token, _, err := PasswordResetRequest(c, "joe@example.com", 1*time.Minute)
	assert.NilError(t, err)

	// verify with token and set new password
	_, err = VerifiedPasswordReset(c, token, "my New PassWord@$1")
	assert.NilError(t, err)

	// check it worked
	cred, err := data.GetCredential(db, data.ByIdentityID(user.ID))
	assert.NilError(t, err)

	err = bcrypt.CompareHashAndPassword(cred.PasswordHash, []byte("my New PassWord@$1"))
	assert.NilError(t, err)

	// check I can't use the token again
	_, err = VerifiedPasswordReset(c, token, "another password")
	assert.Error(t, err, "record not found")
}
