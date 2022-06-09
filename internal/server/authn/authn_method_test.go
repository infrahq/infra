package authn

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
)

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	patch.ModelsSymmetricKey(t)
	db, err := data.NewDB(driver, nil)
	assert.NilError(t, err)

	err = data.CreateProvider(db, &models.Provider{Name: models.InternalInfraProviderName})
	assert.NilError(t, err)

	return db
}

func TestLogin(t *testing.T) {
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
		IdentityID:          user.ID,
		PasswordHash:        hash,
		OneTimePassword:     false,
		OneTimePasswordUsed: false,
	}

	err = data.CreateCredential(db, &creds)
	assert.NilError(t, err)

	t.Run("failed login does not create access key", func(t *testing.T) {
		authn := NewPasswordCredentialAuthentication(username, "invalid password")
		_, bearer, err := Login(db, authn, time.Now().Add(1*time.Minute), time.Minute)

		assert.ErrorContains(t, err, "failed to login")
		assert.Equal(t, bearer, "")
	})

	t.Run("successful login does creates access key for authenticated identity", func(t *testing.T) {
		authn := NewPasswordCredentialAuthentication("gohan@example.com", password)
		exp := time.Now().Add(1 * time.Minute)
		ext := 1 * time.Minute
		key, bearer, err := Login(db, authn, exp, ext)

		assert.NilError(t, err)
		assert.Assert(t, bearer != "")
		assert.Equal(t, key.IssuedFor, user.ID)
		assert.Equal(t, key.ExpiresAt, exp)
		assert.Equal(t, key.Extension, ext)
	})
}
