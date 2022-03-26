package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeRoundTrip(t *testing.T) {
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
			require.NoError(t, err)

			result, err := json.Marshal(tm)
			require.NoError(t, err)

			require.EqualValues(t, test.expected, string(result))
		})
	}
}

func TestDurationRoundTrip(t *testing.T) {
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
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)

			result, err := json.Marshal(tm)
			require.NoError(t, err)

			require.EqualValues(t, test.expected, string(result))
		})
	}
}
