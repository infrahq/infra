package validate

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
)

type ExampleRequest struct {
	RequiredString string `json:"strOne"`
	SubNested      Sub    `json:"subNested"`
	Sub                   // sub embedded
}

type Sub struct {
	FieldOne string `json:"fieldOne"`
}

func (r *ExampleRequest) ValidationRules() []ValidationRule {
	return []ValidationRule{
		Required("strOne", r.RequiredString),
		&StringRule{
			Value:     r.RequiredString,
			Name:      "strOne",
			MinLength: 2,
			MaxLength: 10,
			CharacterRanges: []CharRange{
				AlphabetLower,
				AlphabetUpper,
				Numbers,
				Dot, Dash, Underscore,
			},
		},
		Required("fieldOne", r.Sub.FieldOne),
		&StringRule{
			Value:     r.SubNested.FieldOne,
			Name:      "subNested.fieldOne",
			MaxLength: 10,
		},
	}
}

func TestValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := &ExampleRequest{
			RequiredString: "not-zero",
			Sub:            Sub{FieldOne: "not-zero2"},
		}
		err := Validate(r)
		assert.NilError(t, err)
	})

	t.Run("with failures", func(t *testing.T) {
		r := &ExampleRequest{
			RequiredString: "",
			SubNested: Sub{
				FieldOne: "abcdefghijklmnopqrst",
			},
		}
		err := Validate(r)
		assert.ErrorContains(t, err, "validation failed: ")

		var fieldError Error
		assert.Assert(t, errors.As(err, &fieldError))
		expected := Error{
			"fieldOne":           {"a value is required"},
			"strOne":             {"a value is required"},
			"subNested.fieldOne": {"length (20) must be no more than 10"},
		}
		assert.DeepEqual(t, fieldError, expected)
	})
}

type MutualExample struct {
	First  string
	Second bool
	Third  int
}

func (m MutualExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		MutuallyExclusive(
			Field{Name: "first", Value: m.First},
			Field{Name: "second", Value: m.Second},
			Field{Name: "third", Value: m.Third}),
	}
}

func TestMutuallyExclusive_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := MutualExample{First: "value"}
		assert.NilError(t, Validate(e))

		e = MutualExample{}
		assert.NilError(t, Validate(e))

		e = MutualExample{Second: true}
		assert.NilError(t, Validate(e))

		e = MutualExample{Third: 123}
		assert.NilError(t, Validate(e))
	})
	t.Run("with failure two set", func(t *testing.T) {
		e := MutualExample{First: "value", Second: true}
		err := Validate(e)
		assert.Error(t, err, "validation failed: only one of (first, second) can be set")
	})
	t.Run("with failure three set", func(t *testing.T) {
		e := MutualExample{
			First:  "value",
			Second: true,
			Third:  123,
		}
		err := Validate(e)
		assert.Error(t, err, "validation failed: only one of (first, second, third) can be set")
	})
}

type OneOfExample struct {
	First  string
	Second bool
	Third  int
}

func (m OneOfExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		RequireOneOf(
			Field{Name: "first", Value: m.First},
			Field{Name: "second", Value: m.Second},
			Field{Name: "third", Value: m.Third}),
	}
}

func TestRequireOneOf_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := OneOfExample{First: "value"}
		assert.NilError(t, Validate(e))

		e = OneOfExample{Second: true}
		assert.NilError(t, Validate(e))

		e = OneOfExample{Third: 123}
		assert.NilError(t, Validate(e))
	})
	t.Run("with failure", func(t *testing.T) {
		e := OneOfExample{}
		err := Validate(e)
		assert.Error(t, err, "validation failed: one of (first, second, third) is required")
	})
}
