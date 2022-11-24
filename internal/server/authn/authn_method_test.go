package authn

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/database"
	"github.com/infrahq/infra/internal/testing/patch"
	"github.com/infrahq/infra/uid"
)

func setupDB(t *testing.T) *data.Transaction {
	t.Helper()
	patch.ModelsSymmetricKey(t)
	db, err := data.NewDB(data.NewDBOptions{DSN: database.PostgresDriver(t, "_authn").DSN})
	assert.NilError(t, err)
	return txnForTestCase(t, db, db.DefaultOrg.ID)
}

func txnForTestCase(t *testing.T, db *data.DB, orgID uid.ID) *data.Transaction {
	t.Helper()
	tx, err := db.Begin(context.Background(), nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx.WithOrgID(orgID)
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)
	// setup with user/pass login
	// authentication method should not matter for this test
	username := "gohan@example.com"
	user := &models.Identity{Name: username}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NilError(t, err)

	creds := models.Credential{
		IdentityID:      user.ID,
		PasswordHash:    hash,
		OneTimePassword: false,
	}

	err = data.CreateCredential(db, &creds)
	assert.NilError(t, err)

	t.Run("failed login does not create access key", func(t *testing.T) {
		authn := NewPasswordCredentialAuthentication(username, "invalid password")
		result, err := Login(ctx, db, authn, time.Now().Add(1*time.Minute), time.Minute)

		assert.ErrorContains(t, err, "failed to login")
		assert.Equal(t, result.Bearer, "")
	})

	t.Run("successful login does creates access key for authenticated identity", func(t *testing.T) {
		authn := NewPasswordCredentialAuthentication("gohan@example.com", password)
		exp := time.Now().Add(1 * time.Minute)
		ext := 1 * time.Minute
		result, err := Login(ctx, db, authn, exp, ext)
		assert.NilError(t, err)
		assert.Assert(t, result.Bearer != "")
		assert.Equal(t, result.AccessKey.IssuedFor, user.ID)
		assert.Equal(t, result.AccessKey.ExpiresAt, exp)
		assert.Equal(t, result.AccessKey.InactivityExtension, ext)
		assert.Equal(t, result.User.ID, user.ID)
	})
}
