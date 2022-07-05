package models

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strings"
)

type Base64 []byte

func (f Base64) Value() (driver.Value, error) {
	r := base64.StdEncoding.EncodeToString([]byte(f))

	return r, nil
}

func (f *Base64) Scan(v interface{}) error {
	b, err := base64.StdEncoding.DecodeString(string(v.(string)))
	if err != nil {
		return fmt.Errorf("base64 decoding field: %w", err)
	}

	*f = Base64(b)

	return nil
}

func (f Base64) GormDataType() string {
	return "text"
}

type CommaSeparatedStrings []string

func (s CommaSeparatedStrings) Value() (driver.Value, error) {
	return strings.Join([]string(s), ","), nil
}

func (s *CommaSeparatedStrings) Scan(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("expected string type for %v", v)
	}
	parts := strings.Split(str, ",")

	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	*s = CommaSeparatedStrings(parts)

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
