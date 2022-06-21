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
