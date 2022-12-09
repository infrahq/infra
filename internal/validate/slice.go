package validate

import (
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
}

// Slice validates a slice field
func StringSlice(name string, value []string, itemRule StringRule, maxLength int) ValidationRule {
	return StringSliceRule{Name: name, Value: value, ItemRule: itemRule}
}

func (s StringSliceRule) Validate() *Failure {
	if len(s.Value) == 0 {
		return nil
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

// DescribeSchema does nothing, the schema could vary based on the item rule
func (s StringSliceRule) DescribeSchema(_ *openapi3.Schema) {}
