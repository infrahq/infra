package validate

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type IntRule struct {
	// Value to validate
	Value int
	// Name of the field in json.
	Name string

	// Min is the minimum allowed value.
	Min *int
	// Max is the maximum allowed value.
	Max *int
}

func (i IntRule) Validate() *Failure {
	if i.Value == 0 {
		return nil
	}

	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if i.Min != nil && i.Value < *i.Min {
		add("value %d must be at least %d", i.Value, *i.Min)
	}
	if i.Max != nil && i.Value > *i.Max {
		add("value %d must be at most %d", i.Value, *i.Max)
	}

	if len(problems) > 0 {
		return Fail(i.Name, problems...)
	}
	return nil
}

func (i IntRule) DescribeSchema(parent *openapi3.Schema) {
	schema := schemaForProperty(parent, i.Name)
	if i.Min != nil {
		schema.Min = float64Ptr(*i.Min)
	}
	if i.Max != nil {
		schema.Max = float64Ptr(*i.Max)
	}
}

func float64Ptr(v int) *float64 {
	f := float64(v)
	return &f
}

func Int(v int) *int {
	return &v
}
