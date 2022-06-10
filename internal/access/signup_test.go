package access

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestSignupEnabled(t *testing.T) {
	setup := func(t *testing.T) (*gin.Context, *gorm.DB) {
		db := setupDB(t)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)
		_, err := data.InitializeSettings(db)
		assert.NilError(t, err)
		return c, db
	}

	t.Run("Enabled", func(t *testing.T) {
		c, _ := setup(t)

		enabled, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, enabled)
	})

	t.Run("DisabledByResources", func(t *testing.T) {
		c, db := setup(t)

		err := data.CreateIdentity(db, &models.Identity{Name: "test"})
		assert.NilError(t, err)

		enabled, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, !enabled)
	})

	t.Run("DisabledByDeletedResources", func(t *testing.T) {
		c, db := setup(t)

		enabled, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, enabled)

		err = data.CreateIdentity(db, &models.Identity{Name: "test"})
		assert.NilError(t, err)

		enabled, err = SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, !enabled)

		err = data.DeleteIdentities(db, data.ByName("test"))
		assert.NilError(t, err)

		enabled, err = SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, !enabled)
	})

	t.Run("DisabledByInClusterConnector", func(t *testing.T) {
		c, db := setup(t)

		// emulate an in-cluster connector by installing a connector access key
		identity := models.Identity{Name: models.InternalInfraConnectorIdentityName}

		err := data.CreateIdentity(db, &identity)
		assert.NilError(t, err)

		provider := models.Provider{Name: models.InternalInfraProviderName}

		err = data.CreateProvider(db, &provider)
		assert.NilError(t, err)

		accessKey := models.AccessKey{
			IssuedFor:  identity.ID,
			ProviderID: provider.ID,
			ExpiresAt:  time.Now().Add(time.Minute),
		}

		_, err = data.CreateAccessKey(db, &accessKey)
		assert.NilError(t, err)

		enabled, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Assert(t, !enabled)
	})

	user := "admin@infrahq.com"
	pass := "password"

	t.Run("SignupUser", func(t *testing.T) {
		c, db := setup(t)

		enabled, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Equal(t, enabled, true)

		provider := models.Provider{Name: models.InternalInfraProviderName}

		err = data.CreateProvider(db, &provider)
		assert.NilError(t, err)

		identity, err := Signup(c, user, pass)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)

		enabled, err = SignupEnabled(c)
		assert.NilError(t, err)
		assert.Equal(t, enabled, false)

		// check "admin" user can login
		userPassLogin := authn.NewPasswordCredentialAuthentication(user, pass)
		key, _, requiresUpdate, err := Login(c, userPassLogin, time.Now().Add(time.Hour), time.Hour)
		assert.NilError(t, err)
		assert.Equal(t, identity.ID, key.IssuedFor)
		assert.Equal(t, requiresUpdate, false)

		c.Set("identity", identity)

		// check "admin" can create token
		_, err = CreateToken(c)
		assert.NilError(t, err)
	})
}
