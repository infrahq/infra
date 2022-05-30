package validate

import (
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
)

func TestStringRule_Validate(t *testing.T) {
	t.Run("min length", func(t *testing.T) {
		r := &ExampleRequest{RequiredString: "a"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length (1) must be at least 2")
	})
	t.Run("max length", func(t *testing.T) {
		r := &ExampleRequest{RequiredString: "abcdefghijklm"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length (13) must be no more than 10")
	})
	t.Run("character ranges", func(t *testing.T) {
		r := &ExampleRequest{RequiredString: "almost~valid"}
		err := Validate(r)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"fieldOne": {"a value is required"},
			"strOne": {
				"length (12) must be no more than 10",
				"character ~ at position 6 is not allowed",
			},
		}
		assert.DeepEqual(t, verr, expected)
	})
}

func TestStringRule_DescribeSchema(t *testing.T) {
	r := StringRule{
		Name:      "street",
		MinLength: 2,
		MaxLength: 10,
		CharacterRanges: []CharRange{
			AlphabetUpper,
			Dot, Dash,
		},
	}

	var schema openapi3.Schema
	r.DescribeSchema(&schema)

	var max uint64 = 10
	expected := openapi3.Schema{
		Properties: openapi3.Schemas{
			"street": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					MinLength: 2,
					MaxLength: &max,
					Format:    `[A-Z.\-]`,
				},
			},
		},
	}
	assert.DeepEqual(t, schema, expected, cmpSchema)
}

var cmpSchema = cmpopts.IgnoreUnexported(openapi3.Schema{})
