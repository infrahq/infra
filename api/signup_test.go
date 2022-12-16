package api

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/validate"
)

func TestSignupWithReservedDomain(t *testing.T) {
	req := SignupRequest{
		User: &SignupUser{
			UserName: "foo@example.com",
			Password: "abcdef1235464$!",
		},
		OrgName:   "Foo",
		Subdomain: "infrahq",
	}

	err := validate.Validate(req)
	assert.Error(t, err, "validation failed: subDomain: infrahq is reserved and can not be used")

	req.Subdomain = "zzz"
	err = validate.Validate(req)
	assert.Error(t, err, "validation failed: subDomain: must be at least 4 characters")
}
