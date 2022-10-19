package validate

import (
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type DateRule struct {
	// Value to validate
	Value time.Time
	// Name of the field in json.
	Name string

	NotBefore time.Time
	NotAfter  time.Time
}

func Date(name string, value time.Time, notBefore, notAfter time.Time) DateRule {
	return DateRule{
		Name:      name,
		Value:     value,
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
}

func (s DateRule) DescribeSchema(parent *openapi3.Schema) {
	// schema := schemaForProperty(parent, s.Name)
}

func (s DateRule) Validate() *Failure {
	value := s.Value
	if value.IsZero() {
		return nil
	}

	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}
	if value.Before(s.NotBefore) {
		add("must be after %s", s.NotBefore)
	}
	if value.After(s.NotAfter) {
		add("must be before %s", s.NotAfter)
	}

	if len(problems) > 0 {
		return Fail(s.Name, problems...)
	}
	return nil
}
