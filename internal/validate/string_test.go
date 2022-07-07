package validate

import (
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
)

type StringExample struct {
	Field string
}

func (s StringExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		&StringRule{
			Value:           s.Field,
			Name:            "strField",
			MinLength:       2,
			MaxLength:       10,
			CharacterRanges: []CharRange{AlphabetLower},
		},
	}
}

func TestStringRule_Validate(t *testing.T) {
	t.Run("min length", func(t *testing.T) {
		r := StringExample{Field: "a"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length of string (1) must be at least 2")
	})
	t.Run("max length", func(t *testing.T) {
		r := StringExample{Field: "abcdefghijklm"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length of string (13) must be no more than 10")
	})
	t.Run("character ranges", func(t *testing.T) {
		r := StringExample{Field: "almost~valid"}
		err := Validate(r)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"strField": {
				"length of string (12) must be no more than 10",
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
