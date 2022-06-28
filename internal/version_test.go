package internal

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestFullVersion(t *testing.T) {
	type version struct {
		Version    string
		Prerelease string
		Metadata   string
		Expected   string
	}

	cases := []version{
		{
			Version:  "0.1.0",
			Expected: "0.1.0",
		},
		{
			Version:  "0.1.0-beta",
			Expected: "0.1.0-beta",
		},
		{
			Version:    "0.1.0",
			Prerelease: "beta",
			Expected:   "0.1.0-beta",
		},
		{
			Version:    "0.1.0-beta",
			Prerelease: "alpha",
			Expected:   "0.1.0-beta",
		},
		{
			Version:  "0.1.0+dev",
			Expected: "0.1.1+dev",
		},
		{
			Version:  "0.1.0",
			Metadata: "dev",
			Expected: "0.1.1+dev",
		},
		{
			Version:  "0.1.0+dev",
			Metadata: "123",
			Expected: "0.1.1+dev",
		},
		{
			Version:    "0.1.0",
			Prerelease: "beta",
			Metadata:   "dev",
			Expected:   "0.1.1-beta+dev",
		},
	}

	for _, c := range cases {
		t.Run(c.Expected, func(t *testing.T) {
			Version = c.Version
			Prerelease = c.Prerelease
			Metadata = c.Metadata
			assert.Equal(t, FullVersion(), c.Expected)
		})
	}
}
