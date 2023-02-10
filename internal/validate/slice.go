package validate

import (
	"fmt"

	"github.com/infrahq/infra/internal/openapi3"
)

type SliceRule struct {
	// Value to validate
	Value []string
	// Name of the field in json.
	Name string
	// A rule to apply to each value of the slice
	ItemRule StringRule
}

func (s SliceRule) Validate() *Failure {
	if len(s.Value) == 0 {
		return nil
	}

	for i, val := range s.Value {
		s.ItemRule.Value = val
		if fail := s.ItemRule.Validate(); fail != nil {
			fail.Name = fmt.Sprintf("%v.%d", fail.Name, i+1)
			return fail
		}
	}

	return nil
}

// DescribeSchema does nothing, the schema could vary based on the item rule
func (s SliceRule) DescribeSchema(_ *openapi3.Schema) {}
