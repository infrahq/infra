package api

import (
	"encoding/json"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestTime_MarshalJSON_RoundTripProperty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "nulls",
			input:    `{"T1":null,"T2":null}`,
			expected: `{"T1":null,"T2":null}`,
		},
		{
			name:     "blanks",
			input:    `{"T1":"","T2":""}`,
			expected: `{"T1":null,"T2":null}`,
		},
		{
			name:     "empty",
			input:    `{}`,
			expected: `{"T1":null,"T2":null}`,
		},
		{
			name:     "values",
			input:    `{"T1":"2016-01-02T01:24:21Z","T2":"2016-01-02T01:24:21Z"}`,
			expected: `{"T1":"2016-01-02T01:24:21Z","T2":"2016-01-02T01:24:21Z"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tm := &struct {
				T1 Time
				T2 *Time
			}{}
			err := json.Unmarshal([]byte(test.input), &tm)
			assert.NilError(t, err)

			result, err := json.Marshal(tm)
			assert.NilError(t, err)

			assert.Equal(t, test.expected, string(result))
		})
	}
}

func TestTime_MarshalJSON(t *testing.T) {
	type testCase struct {
		name     string
		source   interface{}
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual, err := json.Marshal(tc.source)
		assert.NilError(t, err)
		assert.Equal(t, tc.expected, string(actual))
	}

	td := time.Date(2020, 1, 2, 3, 4, 5, 6, time.UTC)

	type Container struct {
		Time Time
	}
	type PtrContainer struct {
		Time *Time
	}
	type OmitEmpty struct {
		Time *Time `json:",omitempty"`
	}

	testCases := []testCase{
		{
			name:     "from value",
			source:   Time(td),
			expected: `"2020-01-02T03:04:05Z"`,
		},
		{
			name:     "from pointer",
			source:   (*Time)(&td),
			expected: `"2020-01-02T03:04:05Z"`,
		},
		{
			name:     "from value in struct",
			source:   Container{Time: Time(td)},
			expected: `{"Time":"2020-01-02T03:04:05Z"}`,
		},
		{
			name:     "from value in pointer to struct",
			source:   &Container{Time: Time(td)},
			expected: `{"Time":"2020-01-02T03:04:05Z"}`,
		},
		{
			name:     "from pointer in pointer to struct",
			source:   &PtrContainer{Time: (*Time)(&td)},
			expected: `{"Time":"2020-01-02T03:04:05Z"}`,
		},
		{
			name:     "nil pointer",
			source:   (*Time)(nil),
			expected: `null`,
		},
		{
			name:     "nil pointer in struct",
			source:   &PtrContainer{},
			expected: `{"Time":null}`,
		},
		{
			name:     "nil pointer in struct with omitempty",
			source:   &OmitEmpty{},
			expected: `{}`,
		},
		{
			name:     "with non-UTC location",
			source:   Time(time.Date(2020, 1, 2, 3, 4, 5, 6, time.FixedZone("ET", -5))),
			expected: `"2020-01-02T03:04:10Z"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestDuration_MarshalJSON_RoundTripProperty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		err      string
	}{
		{
			name:     "nulls",
			input:    `{"D2":null}`,
			expected: `{"D1":"0s","D2":null}`,
		},
		{
			name:  "blanks",
			input: `{"D1":"","D2":""}`,
			err:   `invalid duration ""`,
		},
		{
			name:     "empty",
			input:    `{}`,
			expected: `{"D1":"0s","D2":null}`,
		},
		{
			name:     "values",
			input:    `{"D1":"4h0m12s","D2":"4h0m12s"}`,
			expected: `{"D1":"4h0m12s","D2":"4h0m12s"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tm := &struct {
				D1 Duration
				D2 *Duration
			}{}
			err := json.Unmarshal([]byte(test.input), &tm)

			if test.err != "" {
				assert.ErrorContains(t, err, test.err)
				return
			}
			assert.NilError(t, err)

			result, err := json.Marshal(tm)
			assert.NilError(t, err)

			assert.Equal(t, test.expected, string(result))
		})
	}
}
