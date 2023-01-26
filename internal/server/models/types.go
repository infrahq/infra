package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type CommaSeparatedStrings []string

func (s CommaSeparatedStrings) Value() (driver.Value, error) {
	for _, v := range s {
		if strings.Contains(v, ",") {
			return nil, fmt.Errorf("can not store values that include commas")
		}
	}
	return strings.Join(s, ","), nil
}

func (s *CommaSeparatedStrings) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("expected string type for comma separated string, got %T", v)
	}
	parts := strings.Split(str, ",")

	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	*s = parts
	return nil
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

// JSONB represents a JSON binary stored in Postgres
type JSONB []string

func (j JSONB) Value() (driver.Value, error) {
	marshalled, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("convert object to json")
	}
	return string(marshalled), nil
}

// Scan implements the sql.Scanner interface.
func (j *JSONB) Scan(src interface{}) error {
	s, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("cannot scan values which are not a byte array to a JSON blob")
	}
	return json.Unmarshal([]byte(s), &j)
}
