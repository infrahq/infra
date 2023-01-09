package email

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestBuildNameFromEmail(t *testing.T) {
	type testCase struct {
		Email        string
		ExpectedName string
	}
	cases := []testCase{
		{
			Email:        "bruce@example.com",
			ExpectedName: "Bruce",
		},
		{
			Email:        "joe.average@example.com",
			ExpectedName: "Joe",
		},
	}

	for _, c := range cases {
		assert.Equal(t, BuildNameFromEmail(c.Email), c.ExpectedName)
	}
}
