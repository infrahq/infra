package models

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
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
