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
			Value:               s.Field,
			Name:                "strField",
			MinLength:           2,
			MaxLength:           10,
			CharacterRanges:     []CharRange{AlphabetLower, AlphabetUpper},
			FirstCharacterRange: []CharRange{AlphabetLower},
		},
	}
}

func TestStringRule_Validate(t *testing.T) {
	t.Run("min length", func(t *testing.T) {
		r := StringExample{Field: "a"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length of string is 1, must be at least 2")
	})
	t.Run("max length", func(t *testing.T) {
		r := StringExample{Field: "abcdefghijklm"}
		err := Validate(r)
		assert.ErrorContains(t, err, "length of string is 13, must be no more than 10")
	})
	t.Run("character ranges", func(t *testing.T) {
		r := StringExample{Field: "almost~valid"}
		err := Validate(r)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"strField": {
				"length of string is 12, must be no more than 10",
				"character '~' at position 6 is not allowed",
			},
		}
		assert.DeepEqual(t, verr, expected)
	})
	t.Run("character ranges whitespace", func(t *testing.T) {
		r := StringExample{Field: "almost valid"}
		err := Validate(r)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"strField": {
				"length of string is 12, must be no more than 10",
				`character ' ' at position 6 is not allowed`,
			},
		}
		assert.DeepEqual(t, verr, expected)
	})
	t.Run("first character range", func(t *testing.T) {
		r := StringExample{Field: "NotValid"}
		err := Validate(r)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"strField": {"first character 'N' is not allowed"},
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

type EnumExample struct {
	Kind string
}

func (e EnumExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		Enum("kind", e.Kind, []string{"fruit", "legume", "grain"}),
	}
}

func TestEnum_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := EnumExample{}
		assert.NilError(t, Validate(e))

		e = EnumExample{Kind: "grain"}
		assert.NilError(t, Validate(e))

		e = EnumExample{Kind: "legume"}
		assert.NilError(t, Validate(e))

		e = EnumExample{Kind: "fruit"}
		assert.NilError(t, Validate(e))
	})

	t.Run("failure", func(t *testing.T) {
		e := EnumExample{Kind: "mushroom"}
		err := Validate(e)

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"kind": {"must be one of (fruit, legume, grain)"},
		}
		assert.DeepEqual(t, verr, expected)
	})
}

func TestEnum_DescribeSchema(t *testing.T) {
	e := Enum("kind", "", []string{"car", "truck", "bus"})

	var schema openapi3.Schema
	e.DescribeSchema(&schema)

	expected := openapi3.Schema{
		Properties: openapi3.Schemas{
			"kind": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Enum: []interface{}{"car", "truck", "bus"},
				},
			},
		},
	}
	assert.DeepEqual(t, schema, expected, cmpSchema)
}

type ReservedExample struct {
	Value string
}

func (e ReservedExample) ValidationRules() []ValidationRule {
	words := []string{"cars", "boats", "vans"}
	return []ValidationRule{
		ReservedStrings("name", e.Value, words),
	}
}

func TestReserved_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := Validate(ReservedExample{Value: "ok"})
		assert.NilError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		err := Validate(ReservedExample{Value: "cars"})

		var verr Error
		assert.Assert(t, errors.As(err, &verr), "wrong type %T", err)
		expected := Error{
			"name": {"cars is reserved and can not be used"},
		}
		assert.DeepEqual(t, verr, expected)
	})
}
