package cmd

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestProcessRunning(t *testing.T) {
	type testCase struct {
		pid      int
		expected bool
	}

	testCases := []testCase{
		{pid: os.Getpid(), expected: true},
		{pid: -1, expected: false}, // invalid pid
		{pid: 0, expected: false},  // default config pid, also invalid
	}

	for _, tc := range testCases {
		result, err := processRunning(int32(tc.pid))

		assert.NilError(t, err)
		assert.Equal(t, result, tc.expected)
	}
}
