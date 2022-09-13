package api

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/validate"
)

func TestSignupWithReservedDomain(t *testing.T) {
	req := SignupRequest{
		Name:     "foo@example.com",
		Password: "abcdef1235464$!",
		Org: SignupOrg{
			Name:      "Foo",
			Subdomain: "infrahq",
		},
	}

	err := validate.Validate(req)
	assert.Error(t, err, "validation failed: org.subDomain: infrahq is reserved and can not be used")

	req.Org.Subdomain = "zzz"
	err = validate.Validate(req)
	assert.Error(t, err, "validation failed: org.subDomain: must be at least 4 characters")
}
