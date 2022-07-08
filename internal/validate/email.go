package validate

import (
	"fmt"
	"net/mail"

	"github.com/getkin/kin-openapi/openapi3"
)

// Email validates a field that should contain an email address.
func Email(name string, value string) ValidationRule {
	return email{name: name, value: value}
}

type email struct {
	name  string
	value string
}

func (e email) Validate() *Failure {
	if e.value == "" {
		return nil
	}
	addr, err := mail.ParseAddress(e.value)
	if err != nil {
		return fail(e.name, "invalid email address")
	}
	if addr.Name != "" {
		return fail(e.name, fmt.Sprintf("email address must not contain display name %q", addr.Name))
	}
	return nil
}

func (e email) DescribeSchema(parent *openapi3.Schema) {
	schema := schemaForProperty(parent, e.name)
	schema.Format = "email"
}
