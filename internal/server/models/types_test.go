package models

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestCommaSeparatedStringsValue(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{""}, ""},
		{[]string{"one"}, "one"},
		{[]string{"one", "two"}, "one,two"},
	}
	for _, test := range tests {
		val, err := CommaSeparatedStrings(test.input).Value()
		assert.NilError(t, err)
		assert.Equal(t, test.expected, val)
	}

	t.Run("invalid input", func(t *testing.T) {
		input := []string{"ok", "with, comma"}
		_, err := CommaSeparatedStrings(input).Value()
		assert.Error(t, err, "can not store values that include commas")
	})
}

func TestCommaSeparatedStringsScan(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"one", []string{"one"}},
		{"one,two", []string{"one", "two"}},
	}
	for _, test := range tests {
		s := CommaSeparatedStrings([]string{})
		err := s.Scan(test.input)
		assert.NilError(t, err)
		assert.DeepEqual(t, test.expected, ([]string)(s))
	}
}
