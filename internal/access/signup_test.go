package access

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: move this test coverage to the API handler
func TestSignup(t *testing.T) {
	setup := func(t *testing.T) (*gin.Context, data.GormTxn) {
		db := setupDB(t)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = (&http.Request{}).WithContext(context.Background())
		c.Set("db", db)
		return c, db
	}

	user := "admin@example.com"
	pass := "password"
	org := &models.Organization{Name: "acme", Domain: "acme.example.com"}

	signupDetails := SignupDetails{
		Name:     user,
		Password: pass,
		Org:      org,
	}

	t.Run("SignupNewOrg", func(t *testing.T) {
		c, db := setup(t)

		identity, bearer, err := Signup(c, time.Now().Add(1*time.Minute), "example.com", signupDetails)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)
		assert.Equal(t, identity.OrganizationID, org.ID)

		// simulate a request
		tx := data.NewTransaction(db.GormDB(), org.ID)
		c.Set("db", tx)

		// check "admin" user can login
		userPassLogin := authn.NewPasswordCredentialAuthentication(user, pass)
		key, _, requiresUpdate, err := Login(c, userPassLogin, time.Now().Add(time.Hour), time.Hour)
		assert.NilError(t, err)
		assert.Equal(t, identity.ID, key.IssuedFor)
		assert.Assert(t, bearer != "")
		assert.Equal(t, requiresUpdate, false)

		rCtx := RequestContext{
			Authenticated: Authenticated{User: identity},
			DBTxn:         tx,
		}

		// check "admin" can create token
		_, err = CreateToken(rCtx)
		assert.NilError(t, err)
	})
}
