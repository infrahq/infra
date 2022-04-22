package access

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
)

func TestSignup(t *testing.T) {
	setup := func(t *testing.T, signupEnabled bool) *gin.Context {
		db := setupDB(t)

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("db", db)

		_, err := data.InitializeSettings(db, signupEnabled)
		assert.NilError(t, err)

		return c
	}

	user := "admin@infrahq.com"
	pass := "password"

	t.Run("Enabled", func(t *testing.T) {
		c := setup(t, true)

		required, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Equal(t, required, true)

		identity, err := Signup(c, user, pass)
		assert.NilError(t, err)
		assert.Equal(t, identity.Name, user)

		// check "signupEnabled" flag gets flipped
		_, err = Signup(c, user, pass)
		assert.ErrorContains(t, err, "forbidden")

		// check "admin" user can login
		_, identity2, requireUpdate, err := LoginWithUserCredential(c, user, pass, time.Now().Add(time.Hour))
		assert.NilError(t, err)
		assert.DeepEqual(t, identity, identity2)
		assert.Equal(t, requireUpdate, false)

		c.Set("identity", identity2)

		// check "admin" can create token
		_, err = CreateToken(c)
		assert.NilError(t, err)
	})

	t.Run("NotEnabled", func(t *testing.T) {
		c := setup(t, false)

		required, err := SignupEnabled(c)
		assert.NilError(t, err)
		assert.Equal(t, required, false)

		_, err = Signup(c, user, pass)
		assert.ErrorContains(t, err, "forbidden")
	})
}
