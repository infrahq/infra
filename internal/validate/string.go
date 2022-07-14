package validate

import (
	"bytes"
	"fmt"

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
	// must bet within one of these ranges.
	CharacterRanges []CharRange
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
	AlphabetLower = CharRange{Low: 'a', High: 'z'}
	AlphabetUpper = CharRange{Low: 'A', High: 'Z'}
	Numbers       = CharRange{Low: '0', High: '9'}
	Dash          = CharRange{Low: '-', High: '-'}
	Underscore    = CharRange{Low: '_', High: '_'}
	Dot           = CharRange{Low: '.', High: '.'}
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
		add("length of string (%d) must be at least %d", len(value), s.MinLength)
	}

	if s.MaxLength > 0 && len(value) > s.MaxLength {
		add("length of string (%d) must be no more than %d", len(value), s.MaxLength)
	}

	if len(s.CharacterRanges) > 0 {
		for i, c := range value {
			if !inRange(s.CharacterRanges, c) {
				add("character %c at position %v is not allowed", c, i)
				break
			}
		}
	}

	if len(problems) > 0 {
		return fail(s.Name, problems...)
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
