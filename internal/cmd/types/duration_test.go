package types

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestDuration_Set(t *testing.T) {
	type testCase struct {
		source      string
		expected    time.Duration
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		d := Duration(0)
		err := d.Set(tc.source)
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}

		assert.NilError(t, err)
		actual := time.Duration(d)
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{
			source:      "",
			expectedErr: `invalid duration ""`,
		},
		{
			source:   "3h2m1s",
			expected: 3*time.Hour + 121*time.Second,
		},
		{
			source:   "3y",
			expected: 26280 * time.Hour,
		},
		{
			source:   "11d",
			expected: 264 * time.Hour,
		},
		{
			source:   "4w",
			expected: 672 * time.Hour,
		},
		{
			source:   "3y2h",
			expected: 26282 * time.Hour,
		},
		{
			source:   "3y2d",
			expected: 26328 * time.Hour,
		},
		{
			source:   "3y2w4d",
			expected: 26712 * time.Hour,
		},
		{
			source:   "3y2w4d5h8m7s",
			expected: 26717*time.Hour + 487*time.Second,
		},
		{
			source:      "3w2y",
			expectedErr: "invalid number of years: 3w2",
		},
		{
			source:   "-7y",
			expected: -61320 * time.Hour,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.source, func(t *testing.T) {
			run(t, tc)
		})
	}
}
