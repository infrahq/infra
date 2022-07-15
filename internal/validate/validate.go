package validate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Validate that the values in the Request struct are valid according to the
// validation rules defined on the struct.
// If validation fails the error will be of type Error.
//
// Validate automatically traverses the fields on the struct. If any of the
// fields are of a type that implement Request, the validation rules of that
// field will be used as well.
func Validate(req Request) error {
	reqV := reflect.Indirect(reflect.ValueOf(req))
	err := validateStruct(reqV)
	if len(err) > 0 {
		return err
	}
	return nil
}

func validateStruct(v reflect.Value) Error {
	err := make(Error)

	req, ok := v.Interface().(Request)
	if ok && (v.Kind() != reflect.Pointer || !v.IsNil()) {
		for _, rule := range req.ValidationRules() {
			if failure := rule.Validate(); failure != nil {
				err[failure.Name] = append(err[failure.Name], failure.Problems...)
			}
		}
	}

	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if v.Type().Field(i).Anonymous {
				// validate the embedded struct
				for k, v := range validateStruct(f) {
					err[k] = append(err[k], v...)
				}
				continue
			}
			name := fieldName(v.Type().Field(i))
			for k, v := range validateStruct(f) {
				n := name
				if k != "" {
					n = name + "." + k
				}
				err[n] = append(err[n], v...)
			}
		}
	}
	return err
}

// ValidationRule performs validation on one or more struct fields and can
// describe the validation for public API documentation.
type ValidationRule interface {
	// Validate should return nil if the validation passes. If the validation
	// fails the Failure should contain the name of the field and the list of
	// problems.
	Validate() *Failure

	// DescribeSchema should update schema to describe the values that are
	// allowed by the validation. The schema is the parent schema of the request.
	// To set the schema of a property, look it up from schema.Properties[name].
	DescribeSchema(schema *openapi3.Schema)
}

// Failure describes a validation failures.
type Failure struct {
	// Name of the field. The name should be the user visible name as it appears
	// in API documentation (often the json field name, or query parameter),
	// not the name of the struct field.
	Name string
	// Problems is a list of messages that describe the validation failure. They
	// will be part of the API response.
	Problems []string
}

// Request is implemented by all request structs.
type Request interface {
	ValidationRules() []ValidationRule
}

// Error is a map of field names to errors associated with those fields. Errors
// that are associated with the struct or multiple fields will have a key of
// "".
type Error map[string][]string

func (e Error) Error() string {
	var buf strings.Builder
	buf.WriteString("validation failed: ")
	i := 0
	for k, v := range e {
		if i != 0 {
			buf.WriteString(", ")
		}
		i++
		if k == "" {
			buf.WriteString(strings.Join(v, ", "))
			continue
		}
		buf.WriteString(k + ": " + strings.Join(v, ", "))
	}
	return buf.String()
}

func fail(name string, problems ...string) *Failure {
	return &Failure{Name: name, Problems: problems}
}

type requiredRule struct {
	name  string
	value any
}

// Required checks that the value does not have a zero value.
// Zero values are nil, "", 0, false, empty map, empty slice, or the zero value of
// a struct.
// Name is the name of the field as visible to the user, often the json field
// name.
func Required(name string, value any) ValidationRule {
	return requiredRule{name: name, value: value}
}

func (r requiredRule) DescribeSchema(schema *openapi3.Schema) {
	schema.Required = append(schema.Required, r.name)
}

func (r requiredRule) Validate() *Failure {
	if !reflect.ValueOf(r.value).IsZero() {
		return nil
	}
	return fail(r.name, "is required")
}

// Field is used to construct validation rules that incorporate multiple fields.
type Field struct {
	Name  string
	Value interface{}
}

// MutuallyExclusive returns a validation rule that checks that at most one of
// the fields is set to a non-zero value.
func MutuallyExclusive(fields ...Field) ValidationRule {
	return mutuallyExclusive(fields)
}

type mutuallyExclusive []Field

func (m mutuallyExclusive) Validate() *Failure {
	var nonZero []string
	for _, field := range m {
		if !reflect.ValueOf(field.Value).IsZero() {
			nonZero = append(nonZero, field.Name)
		}
	}

	if len(nonZero) > 1 {
		return fail("", fmt.Sprintf("only one of (%v) can be set", strings.Join(nonZero, ", ")))
	}
	return nil
}

// TODO: use oneOf to DescribeSchema
func (m mutuallyExclusive) DescribeSchema(_ *openapi3.Schema) {}

// RequireAnyOf returns a validation rule that checks that at least one of the
// fields is set to a non-zero value.
func RequireAnyOf(fields ...Field) ValidationRule {
	return requireAnyOf(fields)
}

type requireAnyOf []Field

func (m requireAnyOf) Validate() *Failure {
	var zero []string
	for _, field := range m {
		if reflect.ValueOf(field.Value).IsZero() {
			zero = append(zero, field.Name)
		}
	}

	if len(zero) == len(m) {
		return fail("", fmt.Sprintf("one of (%v) is required", strings.Join(zero, ", ")))
	}
	return nil
}

// TODO: use anyOf to DescribeSchema
func (m requireAnyOf) DescribeSchema(_ *openapi3.Schema) {}

// RequireOneOf returns a validation rule that checks that exactly one of the
// fields is set to a non-zero value.
func RequireOneOf(fields ...Field) ValidationRule {
	return requireOneOf(fields)
}

type requireOneOf []Field

func (m requireOneOf) Validate() *Failure {
	var zero []string
	var nonZero []string
	for _, field := range m {
		if reflect.ValueOf(field.Value).IsZero() {
			zero = append(zero, field.Name)
		} else {
			nonZero = append(nonZero, field.Name)
		}
	}

	if len(nonZero) > 1 {
		return fail("", fmt.Sprintf("only one of (%v) can be set", strings.Join(nonZero, ", ")))
	}
	if len(zero) == len(m) {
		return fail("", fmt.Sprintf("one of (%v) is required", strings.Join(zero, ", ")))
	}
	return nil
}

// TODO: use oneOf to DescribeSchema
func (m requireOneOf) DescribeSchema(_ *openapi3.Schema) {}

func schemaForProperty(parent *openapi3.Schema, prop string) *openapi3.Schema {
	if parent.Properties == nil {
		parent.Properties = make(openapi3.Schemas)
	}
	if parent.Properties[prop] == nil {
		parent.Properties[prop] = &openapi3.SchemaRef{Value: &openapi3.Schema{}}
	}
	return parent.Properties[prop].Value
}

func fieldName(f reflect.StructField) string {
	if name, ok := f.Tag.Lookup("form"); ok {
		return name
	}

	if name, ok := f.Tag.Lookup("uri"); ok {
		return name
	}

	// lookup json tag last, as a field may have a uri or form name, but a
	// json name of "-".
	if name, ok := f.Tag.Lookup("json"); ok {
		name = strings.Split(name, ",")[0]
		if name == "-" {
			return ""
		}
		return name
	}

	return ""
}
