package validate

import (
	"errors"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"gotest.tools/v3/assert"
)

type IntExample struct {
	Start int
	End   int
}

func (i IntExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		IntRule{
			Value: i.Start,
			Name:  "start",
			Min:   Int(-2),
			Max:   Int(20),
		},
		IntRule{
			Value: i.End,
			Name:  "end",
			Min:   Int(0),
			Max:   Int(55),
		},
	}
}

func TestIntRule_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		i := IntExample{}
		assert.NilError(t, Validate(i))

		i = IntExample{Start: -2, End: 55}
		assert.NilError(t, Validate(i))

		i = IntExample{Start: -1}
		assert.NilError(t, Validate(i))

		i = IntExample{End: 20}
		assert.NilError(t, Validate(i))
	})

	t.Run("failure", func(t *testing.T) {
		i := IntExample{Start: -22, End: 60}
		err := Validate(i)
		assert.ErrorContains(t, err, "validation failed")

		var vErr Error
		assert.Assert(t, errors.As(err, &vErr), "wrong type %T", err)
		expected := Error{
			"start": {"value -22 must be at least -2"},
			"end":   {"value 60 must be at most 55"},
		}
		assert.DeepEqual(t, vErr, expected)
	})
}

func TestIntRule_DescribeSchema(t *testing.T) {
	i := IntRule{Name: "count", Min: Int(3), Max: Int(5)}

	var schema openapi3.Schema
	i.DescribeSchema(&schema)

	expected := openapi3.Schema{
		Properties: openapi3.Schemas{
			"count": &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Min: float64Ptr(3),
					Max: float64Ptr(5),
				},
			},
		},
	}
	assert.DeepEqual(t, schema, expected, cmpSchema)
}
