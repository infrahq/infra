package validate

import (
	"errors"
	"testing"

	"gotest.tools/v3/assert"
)

type ExampleRequest struct {
	ID string

	Either string
	Or     int

	First  string
	Second int
	Third  bool

	EmailAddr  string
	EmailOther string

	TooFew    string
	TooMany   string
	WrongOnes string

	TooLow  int
	TooHigh int
}

func (r ExampleRequest) ValidationRules() []ValidationRule {
	return []ValidationRule{
		Required("id", r.ID),
		MutuallyExclusive(
			Field{Name: "either", Value: r.Either},
			Field{Name: "or", Value: r.Or},
		),
		RequireOneOf(
			Field{Name: "first", Value: r.First},
			Field{Name: "second", Value: r.Second},
			Field{Name: "third", Value: r.Third},
		),
		Email("emailAddr", r.EmailAddr),
		Email("emailOther", r.EmailOther),

		StringRule{
			Name:      "tooFew",
			Value:     r.TooFew,
			MinLength: 5,
		},
		StringRule{
			Name:      "tooMany",
			Value:     r.TooMany,
			MaxLength: 5,
		},
		StringRule{
			CharacterRanges: []CharRange{AlphabetLower},
		},

		IntRule{Name: "tooLow", Value: r.TooLow, Min: Int(20)},
		IntRule{Name: "tooHigh", Value: r.TooHigh, Max: Int(20)},
	}
}

func TestValidate_AllRules(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := ExampleRequest{
			ID:         "id",
			First:      "something",
			EmailAddr:  "valid@example.com",
			EmailOther: "other@example.com",
			TooFew:     "abcdef",
			WrongOnes:  "abc",
			TooLow:     22,
		}
		err := Validate(r)
		assert.NilError(t, err)
	})

	t.Run("with failures", func(t *testing.T) {
		r := ExampleRequest{
			Either:     "yes",
			Or:         1,
			EmailAddr:  "nope~example.com",
			EmailOther: `"Display Name" <other@example.com>`,
			TooFew:     "a",
			TooMany:    "ababab",
			WrongOnes:  "ah CAPS",
			TooLow:     2,
			TooHigh:    22,
		}
		err := Validate(r)
		assert.ErrorContains(t, err, "validation failed: ")

		var fieldError Error
		assert.Assert(t, errors.As(err, &fieldError))
		expected := Error{
			"id": {"is required"},
			"": {
				"only one of (either, or) can be set",
				"one of (first, second, third) is required",
			},
			"emailAddr":  {"invalid email address"},
			"emailOther": {`email address must not contain display name "Display Name"`},
			"tooFew":     {"length of string (1) must be at least 5"},
			"tooMany":    {"length of string (6) must be no more than 5"},
			"tooHigh":    {"value (22) must be at most 20"},
			"tooLow":     {"value (2) must be at least 20"},
		}
		assert.DeepEqual(t, fieldError, expected)
	})
}

type NestedExample struct {
	Anything string
	Sub      SubExample `json:"sub"`
	ExampleRequest
}

func (n NestedExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		StringRule{Name: "any", Value: n.Anything, MaxLength: 3},
	}
}

type SubExample struct {
	Ok     bool
	Nested ExampleRequest `json:"nested"`
}

func (s SubExample) ValidationRules() []ValidationRule {
	return []ValidationRule{
		Required("ok", s.Ok),
	}
}

func TestValidate_Nested(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		n := NestedExample{
			Sub: SubExample{
				Ok:     true,
				Nested: ExampleRequest{ID: "id", Third: true},
			},
			ExampleRequest: ExampleRequest{ID: "ok", First: "1"},
		}
		err := Validate(n)
		assert.NilError(t, err)
	})

	t.Run("with failures", func(t *testing.T) {
		n := NestedExample{
			Anything: "abcdef",
			Sub: SubExample{
				Nested: ExampleRequest{
					Either: "yes",
					Or:     1,
				},
			},
			ExampleRequest: ExampleRequest{
				ID:     "ok",
				TooFew: "a",
			},
		}
		err := Validate(n)
		assert.ErrorContains(t, err, "validation failed: ")
		var fieldError Error
		assert.Assert(t, errors.As(err, &fieldError))
		expected := Error{
			"":    {"one of (first, second, third) is required"},
			"any": {"length of string (6) must be no more than 3"},
			"sub.nested": {
				"only one of (either, or) can be set",
				"one of (first, second, third) is required",
			},
			"sub.nested.id": {"is required"},
			"sub.ok":        {"is required"},
			"tooFew":        {"length of string (1) must be at least 5"},
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
