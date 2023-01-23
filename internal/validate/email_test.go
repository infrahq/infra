package validate

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/openapi3"
)

type EmailExample struct {
	Address string
}

func (e EmailExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		Email("addr", e.Address),
	}
}

func TestEmail_Validate(t *testing.T) {
	type testCase struct {
		name        string
		email       string
		expectedErr string
	}

	run := func(t *testing.T, tc testCase) {
		e := EmailExample{Address: tc.email}
		err := Validate(e)
		if tc.expectedErr == "" {
			assert.NilError(t, err)
			return
		}
		assert.ErrorContains(t, err, tc.expectedErr)
	}

	var testCases = []testCase{
		{
			name:  "standard",
			email: "myaddr@extra.example.com",
		},
		{
			name:  "short",
			email: "m@e.tv",
		},
		{
			name:  "with angle brackets",
			email: "<myaddr@example.com>",
		},
		{
			name:  "starts with number",
			email: "91ok@example.com",
		},
		{
			name:        "with name",
			email:       "My Name <myaddr@example.com>",
			expectedErr: `validation failed: addr: email address must not contain display name "My Name"`,
		},
		{
			name:        "too many ats",
			email:       "foo@what@example.com",
			expectedErr: "validation failed: addr: invalid email address",
		},
		{
			name:        "missing username",
			email:       "@example.com",
			expectedErr: "validation failed: addr: invalid email address",
		},
		{
			name:        "no hostname",
			email:       "sam@",
			expectedErr: "validation failed: addr: invalid email address",
		},
		{
			name:        "missing at",
			email:       "james",
			expectedErr: "validation failed: addr: invalid email address",
		},
		{
			name:        "missing domain '.'",
			email:       "james@example",
			expectedErr: "validation failed: addr: email address must contain at least one '.' character",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestEmail_DescribeSchema(t *testing.T) {
	e := Email("addrField", "")
	var schema openapi3.Schema
	e.DescribeSchema(&schema)
	expected := openapi3.Schema{
		Properties: map[string]*openapi3.SchemaRef{
			"addrField": &openapi3.SchemaRef{
				Schema: &openapi3.Schema{Format: "email"},
			},
		},
	}
	assert.DeepEqual(t, schema, expected, cmpSchema)
}
