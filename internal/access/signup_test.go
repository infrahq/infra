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

func getDefaultOrg(db *gorm.DB) *models.Organization {
	org, ok := db.Statement.Context.Value(data.OrgCtxKey{}).(*models.Organization)
	if !ok {
		panic("org missing from db context")
	}
	return org
}

func TestSignup(t *testing.T) {
	setup := func(t *testing.T) (*gin.Context, *gorm.DB) {
		db := setupDB(t)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = (&http.Request{}).WithContext(context.Background())
		c.Set("db", db)
		_, err := data.InitializeSettings(db, getDefaultOrg(db))
		assert.NilError(t, err)
		return c, db
	}

	user := "admin@infrahq.com"
	pass := "password"

	t.Run("SignupNewOrg", func(t *testing.T) {
		c, db := setup(t)

		identity, createdOrg, err := Signup(c, "acme", "acme.infrahq.com", user, pass)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)

		assert.Equal(t, identity.OrganizationID, createdOrg.ID)

		// check "admin" user can login
		userPassLogin := authn.NewPasswordCredentialAuthentication(user, pass)
		key, _, requiresUpdate, err := Login(c, userPassLogin, time.Now().Add(time.Hour), time.Hour)
		assert.NilError(t, err)
		assert.Equal(t, identity.ID, key.IssuedFor)
		assert.Equal(t, requiresUpdate, false)

		rCtx := RequestContext{
			Authenticated: Authenticated{User: identity},
			DBTxn:         db,
		}

		// check "admin" can create token
		_, err = CreateToken(rCtx)
		assert.NilError(t, err)
	})
}
