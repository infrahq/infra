package types

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestHostPort(t *testing.T) {
	type testCase struct {
		name        string
		input       string
		expectedErr string
		expected    HostPort
	}

	run := func(t *testing.T, tc testCase) {
		hp := HostPort{}
		err := hp.Set(tc.input)

		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}
		assert.NilError(t, err)
		assert.DeepEqual(t, hp, tc.expected)
	}

	testCases := []testCase{
		{
			name:     "ipv4 address",
			input:    "10.10.10.1",
			expected: HostPort{Host: "10.10.10.1"},
		},
		{
			name:     "ipv4 with port",
			input:    "10.10.10.1:1010",
			expected: HostPort{Host: "10.10.10.1", Port: 1010},
		},
		{
			name:     "hostname",
			input:    "example.com",
			expected: HostPort{Host: "example.com"},
		},
		{
			name:     "hostname with port",
			input:    "example.com:8080",
			expected: HostPort{Host: "example.com", Port: 8080},
		},
		{
			name:     "ipv6 address",
			input:    "[aa00:aa00:aa00:aa00:aa00:aa00]",
			expected: HostPort{Host: "[aa00:aa00:aa00:aa00:aa00:aa00]"},
		},
		{
			name:     "ipv6 address with port",
			input:    "[aa00:aa00:aa00:aa00:aa00:aa00]:8080",
			expected: HostPort{Host: "aa00:aa00:aa00:aa00:aa00:aa00", Port: 8080},
		},
		{
			name:        "invalid input",
			input:       "10:10:10",
			expectedErr: "too many colons in address",
		},
		{
			name:        "invalid port",
			input:       "localhost:dog",
			expectedErr: `port "dog" must be a number`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
