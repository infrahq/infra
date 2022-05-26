package cmd

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestProcessRunning(t *testing.T) {
	type testCase struct {
		pid      int
		err      string
		expected bool
	}

	testCases := []testCase{
		{pid: os.Getpid(), err: "", expected: true},
		{pid: -1, err: "invalid pid", expected: false}, // invalid pid
		{pid: 0, err: "", expected: false},             // default config pid, also invalid
	}

	for _, tc := range testCases {
		result, err := processRunning(int32(tc.pid))

		if tc.err != "" {
			assert.ErrorContains(t, err, tc.err)
		} else {
			assert.NilError(t, err)
		}
		assert.Equal(t, result, tc.expected)
	}
}
