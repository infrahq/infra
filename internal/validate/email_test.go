package validate

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"gotest.tools/v3/assert"
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
			name:  "with angles",
			email: "<myaddr@example.com>",
		},
		{
			name:        "no tld",
			email:       "sam@",
			expectedErr: "validation failed: addr: invalid email address: no angle-addr",
		},
		{
			name:        "with name",
			email:       "My Name <myaddr@example.com>",
			expectedErr: "validation failed: addr: email address must not contain a name",
		},
		{
			name:        "too many ats",
			email:       "foo@what@example.com",
			expectedErr: "validation failed: addr: invalid email address: expected single address",
		},
		{
			name:        "missing username",
			email:       "@example.com",
			expectedErr: "validation failed: addr: invalid email address: no angle-addr",
		},
		{
			name:        "missing at",
			email:       "james",
			expectedErr: "validation failed: addr: invalid email address: missing '@' or angle-addr",
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
		Properties: openapi3.Schemas{
			"addrField": &openapi3.SchemaRef{
				Value: &openapi3.Schema{Format: "email"},
			},
		},
	}
	assert.DeepEqual(t, schema, expected, cmpSchema)
}
