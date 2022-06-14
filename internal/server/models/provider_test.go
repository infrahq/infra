package models

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseProviderKind(t *testing.T) {
	t.Run("accepts a valid provider kind", func(t *testing.T) {
		kind, err := ParseProviderKind(OktaKind.String())
		assert.NilError(t, err)
		assert.Equal(t, OktaKind, kind)
	})
	t.Run("rejects invalid provider kinds", func(t *testing.T) {
		_, err := ParseProviderKind("invalid-provider-kind")
		assert.ErrorContains(t, err, "not a valid provider kind")
	})
	t.Run("defaults to oidc", func(t *testing.T) {
		kind, err := ParseProviderKind("")
		assert.NilError(t, err)
		assert.Equal(t, OIDCKind, kind)
	})
}
