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

	switch v.Kind() { // nolint:exhaustive
	case reflect.Struct:
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
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			for k, v := range validateStruct(v.Index(i)) {
				err[k] = append(err[k], v...)
			}
		}
	}
	return err
}

// ValidationRule performs validation on one or more struct fields and can
// describe the validation for public API documentation.
//
// Validation rules should all default to optional. If the field has a zero value
// then the validation rule will do nothing. Use Required, or RequireOneOf to
// make something a required field.
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
		return fail("", fmt.Sprintf("only one of (%v) can have a value", strings.Join(nonZero, ", ")))
	}
	return nil
}

// DescribeSchema does nothing. There is currently no way clean way to express
// "not set" in the OpenAPI spec.
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

func (m requireAnyOf) DescribeSchema(schema *openapi3.Schema) {
	for _, f := range m {
		schema.AnyOf = append(schema.AnyOf, &openapi3.SchemaRef{
			Value: &openapi3.Schema{Required: []string{f.Name}},
		})
	}
}

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
		return fail("", fmt.Sprintf("only one of (%v) can have a value", strings.Join(nonZero, ", ")))
	}
	if len(zero) == len(m) {
		return fail("", fmt.Sprintf("one of (%v) is required", strings.Join(zero, ", ")))
	}
	return nil
}

func (m requireOneOf) DescribeSchema(schema *openapi3.Schema) {
	for _, f := range m {
		schema.OneOf = append(schema.OneOf, &openapi3.SchemaRef{
			Value: &openapi3.Schema{Required: []string{f.Name}},
		})
	}
}

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

	if f.Name == "" {
		return ""
	}

	return strings.ToLower(f.Name[:1]) + f.Name[1:]
}

// ValidatorFunc wraps a function so that it implements ValidationRule. It can
// be used to create special validations without having to define a type.
// The ValidationRule will have a no-op implementation of DescribeSchema.
type ValidatorFunc func() *Failure

func (f ValidatorFunc) Validate() *Failure {
	return f()
}

func (f ValidatorFunc) DescribeSchema(*openapi3.Schema) {}
