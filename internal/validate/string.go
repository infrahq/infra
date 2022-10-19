package validate

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi3"
)

type StringRule struct {
	// Value to validate
	Value string
	// Name of the field in json.
	Name string

	// MinLength is the minimum allowed length of the string in bytes.
	MinLength int
	// MaxLength is the maximum allowed length of the string in bytes.
	MaxLength int

	// CharacterRanges is a list of character ranges. Every rune in value
	// must be within one of these ranges.
	CharacterRanges []CharRange

	// FirstCharacterRange is a list of character ranges. The first rune in
	// value must be within one of these ranges.
	FirstCharacterRange []CharRange
}

func String(name, value string, minLength, maxLength int, charset []CharRange) StringRule {
	return StringRule{
		Name:            name,
		Value:           value,
		MinLength:       minLength,
		MaxLength:       maxLength,
		CharacterRanges: charset,
	}
}

type CharRange struct {
	Low  rune
	High rune
}

func (r CharRange) String() string {
	if r.Low == r.High {
		if r.Low == '-' {
			return `\-`
		}
		return string(r.Low)
	}
	return string(r.Low) + "-" + string(r.High)
}

var (
	AlphabetLower      = CharRange{Low: 'a', High: 'z'}
	AlphabetUpper      = CharRange{Low: 'A', High: 'Z'}
	Numbers            = CharRange{Low: '0', High: '9'}
	Dash               = CharRange{Low: '-', High: '-'}
	Underscore         = CharRange{Low: '_', High: '_'}
	Dot                = CharRange{Low: '.', High: '.'}
	AtSign             = CharRange{Low: '@', High: '@'}
	AlphaNumeric       = []CharRange{AlphabetLower, AlphabetUpper, Numbers}
	DeviceFlowUserCode = []CharRange{{Low: 'B', High: 'D'}, {Low: 'F', High: 'H'}, {Low: 'J', High: 'N'}, {Low: 'P', High: 'T'}, {Low: 'V', High: 'X'}, {Low: 'Z', High: 'Z'}}
)

func (s StringRule) DescribeSchema(parent *openapi3.Schema) {
	schema := schemaForProperty(parent, s.Name)

	schema.MinLength = uint64(s.MinLength)
	if s.MaxLength > 0 {
		max := uint64(s.MaxLength)
		schema.MaxLength = &max
	}

	var buf bytes.Buffer
	for _, r := range s.CharacterRanges {
		buf.WriteString(r.String())
	}
	if buf.Len() > 0 {
		schema.Format = "[" + buf.String() + "]"
	}
}

func (s StringRule) Validate() *Failure {
	value := s.Value
	if value == "" {
		return nil
	}

	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}
	if s.MinLength > 0 && len(value) < s.MinLength {
		add("must be at least %d characters", s.MinLength)
	}

	if s.MaxLength > 0 && len(value) > s.MaxLength {
		add("can be at most %d characters", s.MaxLength)
	}

	if len(s.FirstCharacterRange) > 0 {
		first, _ := utf8.DecodeRuneInString(value)
		if !inRange(s.FirstCharacterRange, first) {
			add("first character %q is not allowed", first)
		}
	}

	if len(s.CharacterRanges) > 0 {
		for i, c := range value {
			if !inRange(s.CharacterRanges, c) {
				add("character %q at position %v is not allowed", c, i)
				break
			}
		}
	}

	if len(problems) > 0 {
		return Fail(s.Name, problems...)
	}
	return nil
}

func inRange(ranges []CharRange, c rune) bool {
	for _, r := range ranges {
		if c >= r.Low && c <= r.High {
			return true
		}
	}
	return false
}

// Enum returns a validation rule that checks that value is one of the allowed
// strings.
func Enum(name string, value string, allowed []string) ValidationRule {
	return enum{Name: name, Value: value, Allowed: allowed}
}

type enum struct {
	Name    string
	Value   string
	Allowed []string
}

func (e enum) Validate() *Failure {
	if e.Value == "" {
		return nil
	}
	for _, ok := range e.Allowed {
		if e.Value == ok {
			return nil
		}
	}
	msg := fmt.Sprintf("must be one of (%v)", strings.Join(e.Allowed, ", "))
	return Fail(e.Name, msg)
}

func (e enum) DescribeSchema(parent *openapi3.Schema) {
	schema := schemaForProperty(parent, e.Name)
	for _, v := range e.Allowed {
		schema.Enum = append(schema.Enum, v)
	}
}

// ReservedStrings returns a validation that checks that value does not match
// any of the strings in values.
func ReservedStrings(name string, value string, values []string) ValidationRule {
	return reserved{Name: name, Value: value, Reserved: values}
}

type reserved struct {
	Name     string
	Value    string
	Reserved []string
}

func (r reserved) Validate() *Failure {
	if r.Value == "" {
		return nil
	}
	for _, notAllowed := range r.Reserved {
		if r.Value == notAllowed {
			msg := fmt.Sprintf("%v is reserved and can not be used", r.Value)
			return Fail(r.Name, msg)
		}
	}
	return nil
}

func (r reserved) DescribeSchema(_ *openapi3.Schema) {}
