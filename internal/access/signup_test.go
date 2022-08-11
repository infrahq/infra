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
	"github.com/infrahq/infra/internal/server/models"
)

func TestSignup(t *testing.T) {
	setup := func(t *testing.T) (*gin.Context, *gorm.DB) {
		db := setupDB(t)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = (&http.Request{}).WithContext(context.Background())
		c.Set("db", db)
		return c, db
	}

	user := "admin@infrahq.com"
	pass := "password"
	org := &models.Organization{Name: "acme", Domain: "acme.infrahq.com"}

	t.Run("SignupNewOrg", func(t *testing.T) {
		c, db := setup(t)

		identity, bearer, err := Signup(c, time.Now().Add(1*time.Minute), user, pass, org)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)
		assert.Equal(t, identity.OrganizationID, org.ID)

		// check "admin" user can login
		userPassLogin := authn.NewPasswordCredentialAuthentication(user, pass)
		key, _, requiresUpdate, err := Login(c, userPassLogin, time.Now().Add(time.Hour), time.Hour)
		assert.NilError(t, err)
		assert.Equal(t, identity.ID, key.IssuedFor)
		assert.Assert(t, bearer != "")
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
