package data

import (
	"testing"

	"github.com/scim2/filter-parser/v2"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
)

func TestFilterParser(t *testing.T) {
	type testCase struct {
		name          string
		expression    string
		expectedQuery string
		expectedArgs  []any
	}

	testCases := []testCase{
		{
			name:          "equality",
			expression:    "id eq \"a1234\"",
			expectedQuery: " identity_id = ? ",
			expectedArgs:  []any{"a1234"},
		},
		{
			name:          "present",
			expression:    "userName pr",
			expectedQuery: " email IS NOT NULL ",
		},
		{
			name:          "not equal",
			expression:    "email ne \"hello@example.com\"",
			expectedQuery: " email != ? ",
			expectedArgs:  []any{"hello@example.com"},
		},
		{
			name:          "starts with",
			expression:    "name.givenName sw \"S\"",
			expectedQuery: " givenName LIKE ? ",
			expectedArgs:  []any{"S%"},
		},
		{
			name:          "contains",
			expression:    "name.familyName co \"S\"",
			expectedQuery: " familyName LIKE ? ",
			expectedArgs:  []any{"%S%"},
		},
		{
			name:          "ends with",
			expression:    "userName ew \"S\"",
			expectedQuery: " email LIKE ? ",
			expectedArgs:  []any{"%S"},
		},
		{
			name:          "logical and",
			expression:    "(email eq \"M\") and (email eq \"W\")",
			expectedQuery: " email = ? AND email = ? ",
			expectedArgs:  []any{"M", "W"},
		},
		{
			name:          "logical or",
			expression:    "(email eq \"M\") or (email eq \"W\")",
			expectedQuery: " email = ? OR email = ? ",
			expectedArgs:  []any{"M", "W"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp, err := filter.ParseFilter([]byte(tc.expression))
			assert.NilError(t, err)
			query := querybuilder.New("")
			err = filterSQL(exp, query)
			assert.NilError(t, err)
			assert.Equal(t, query.String(), tc.expectedQuery)
			if tc.expectedArgs != nil {
				assert.DeepEqual(t, query.Args, tc.expectedArgs)
			}
		})
	}
}

func TestFilterParserError(t *testing.T) {
	type testCase struct {
		name       string
		expression string
	}

	testCases := []testCase{
		{
			name:       "unsupported operator",
			expression: "id gt 1 EAND id lt 123",
		},
		{
			name:       "attempt to return all",
			expression: "1=1",
		},
		{
			name:       "attempt to return all with comparator",
			expression: "id eq 1 OR 1=1",
		},
		{
			name:       "attempt always true equality",
			expression: "name eq \"test\" OR \" or \"\"=\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := filter.ParseFilter([]byte(tc.expression))
			assert.ErrorContains(t, err, "parse conflict")
		})
	}
}

func TestFilterSQLError(t *testing.T) {
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
			expectedErrMsg: "unsupported comparator",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exp, err := filter.ParseFilter([]byte(tc.expression))
			assert.NilError(t, err)
			query := querybuilder.New("")
			err = filterSQL(exp, query)
			assert.ErrorContains(t, err, tc.expectedErrMsg)
		})
	}
}
