package validate

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type StringSliceRule struct {
	// Value to validate
	Value []string
	// Name of the field in json.
	Name string
	// A rule to apply to each value of the slice
	ItemRule StringRule

	// MaxLength is the maximum allowed length of the slice
	MaxLength int
}

// Slice validates a slice field
func StringSlice(name string, value []string, itemRule StringRule, maxLength int) ValidationRule {
	return StringSliceRule{Name: name, Value: value, ItemRule: itemRule, MaxLength: maxLength}
}

func (s StringSliceRule) Validate() *Failure {
	if len(s.Value) == 0 {
		return nil
	}
	if len(s.Value) > s.MaxLength {
		return Fail(s.Name, fmt.Sprintf("max length of %s is %d", s.Name, s.MaxLength))
	}

	for _, val := range s.Value {
		if strings.ContainsAny(val, ",") {
			return Fail(s.Name, "list values cannot contain commas")
		}
		s.ItemRule.Value = val
		if fail := s.ItemRule.Validate(); fail != nil {
			return fail
		}
	}

	return nil
}

func (s StringSliceRule) DescribeSchema(parent *openapi3.Schema) {
	schema := schemaForProperty(parent, s.Name)
	if s.MaxLength > 0 {
		len := uint64(s.MaxLength)
		schema.MaxItems = &len
	}
}
