package access

import (
	"context"
	"net/http"
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

func TestSignup(t *testing.T) {
	setup := func(t *testing.T) (*gin.Context, *gorm.DB) {
		db := setupDB(t)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = (&http.Request{}).WithContext(context.Background())
		c.Set("db", db)
		_, err := data.InitializeSettings(db)
		assert.NilError(t, err)
		return c, db
	}

	user := "admin@infrahq.com"
	pass := "password"
	org := "acme"

	t.Run("SignupNewOrg", func(t *testing.T) {
		c, db := setup(t)

		identity, err := Signup(c, user, pass, org)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)

		createdOrg, err := data.ListOrganizations(db, &models.Pagination{Page: 1, Limit: 1}, data.ByName(org))
		assert.NilError(t, err)
		assert.Equal(t, identity.OrganizationID, createdOrg[0].ID)

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
