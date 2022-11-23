package email

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDomain(t *testing.T) {
	t.Run("rejects non-email format", func(t *testing.T) {
		_, err := Domain("hello")
		assert.ErrorContains(t, err, "hello is an invalid email address")
	})
	t.Run("gets domain of typical email", func(t *testing.T) {
		domain, err := Domain("hello@example.com")
		assert.NilError(t, err)
		assert.Equal(t, domain, "example.com")
	})
	t.Run("gets last domain in email", func(t *testing.T) {
		domain, err := Domain("hello@example.com@infrahq.com")
		assert.NilError(t, err)
		assert.Equal(t, domain, "infrahq.com")
	})
}
