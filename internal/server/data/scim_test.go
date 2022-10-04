package data

import (
	"testing"

	"github.com/scim2/filter-parser/v2"
	"gotest.tools/v3/assert"
)

func TestFilterParser(t *testing.T) {
	type testCase struct {
		name       string
		expression string
		expected   string
	}

	testCases := []testCase{
		{
			name:       "equality",
			expression: "id eq \"a1234\"",
			expected:   "identity_id = 'a1234'",
		},
		{
			name:       "present",
			expression: "userName pr",
			expected:   "email IS NOT NULL",
		},
		{
			name:       "not equal",
			expression: "email ne \"hello@example.com\"",
			expected:   "email != 'hello@example.com'",
		},
		{
			name:       "starts with",
			expression: "name.givenName sw \"S\"",
			expected:   "givenName LIKE 'S%'",
		},
		{
			name:       "contains",
			expression: "name.familyName co \"S\"",
			expected:   "familyName LIKE '%S%'",
		},
		{
			name:       "ends with",
			expression: "userName ew \"S\"",
			expected:   "email LIKE '%S'",
		},
		{
			name:       "logical and",
			expression: "(email eq \"M\") and (email eq \"W\")",
			expected:   "email = 'M' AND email = 'W'",
		},
		{
			name:       "logical or",
			expression: "(email eq \"M\") or (email eq \"W\")",
			expected:   "email = 'M' OR email = 'W'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp, err := filter.ParseFilter([]byte(tc.expression))
			assert.NilError(t, err)
			result, err := filterSQL(exp)
			assert.NilError(t, err)
			assert.Equal(t, result, tc.expected)
		})
	}
}

func TestFilterParserError(t *testing.T) {
	type testCase struct {
		name           string
		expression     string
		expectedErrMsg string
	}

	testCases := []testCase{
		{
			name:           "unknown attribute",
			expression:     "password eq \"a1234\"",
			expectedErrMsg: "unsupported filter attribute",
		},
		{
			name:           "unsupported comparator",
			expression:     "id lt 123",
			expectedErrMsg: "upsupported comparator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp, err := filter.ParseFilter([]byte(tc.expression))
			assert.NilError(t, err)
			_, err = filterSQL(exp)
			assert.ErrorContains(t, err, tc.expectedErrMsg)
		})
	}
}
