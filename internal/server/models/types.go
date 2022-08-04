package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type CommaSeparatedStrings []string

func (s CommaSeparatedStrings) Value() (driver.Value, error) {
	return strings.Join([]string(s), ","), nil
}

func (s *CommaSeparatedStrings) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("expected string type for %v", v)
	}
	parts := strings.Split(str, ",")

	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	*s = parts

	return nil
}

func (f CommaSeparatedStrings) GormDataType() string {
	return "text"
}

func (s *CommaSeparatedStrings) Includes(str string) bool {
	if s == nil {
		return false
	}

	for _, item := range *s {
		if item == str {
			return true
		}
	}

	return false
}
