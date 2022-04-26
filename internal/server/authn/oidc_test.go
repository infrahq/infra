package authn

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestValidateInvalidURL(t *testing.T) {
	oidc := NewOIDC("invalid.example.com", "some_client_id", "some_client_secret", "http://localhost:8301")

	err := oidc.Validate()
	assert.ErrorIs(t, err, ErrInvalidProviderURL)
}
