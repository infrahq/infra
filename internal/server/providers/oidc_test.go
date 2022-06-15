package providers

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/models"
)

func TestValidateInvalidURL(t *testing.T) {
	oidc := NewOIDC(models.Provider{Name: "example-oidc", Kind: models.OIDCKind, URL: "example.com"}, "some_client_secret", "http://localhost:8301")

	err := oidc.Validate()
	assert.ErrorIs(t, err, ErrInvalidProviderURL)
}

func TestUserInfo(t *testing.T) {
	t.Run("no email and no name fails validation", func(t *testing.T) {
		claims := &InfoClaims{}
		err := claims.validate()
		assert.ErrorContains(t, err, "name or email are required")
	})
	t.Run("groups are not required", func(t *testing.T) {
		claims := &InfoClaims{Email: "hello@example.com"}
		err := claims.validate()
		assert.NilError(t, err)
	})
}
