package data

import (
	"database/sql"
	"database/sql/driver"
)

// optionalString has the behaviour of sql.NullString. A null entry
// in a database column is scanned as the empty string, and an empty string
// is saved as a null. Instead of using sql.NullString as the field,
// optionalString allows us to wrap a regular string field on a struct.
// With optionalString you lose the Valid field which identifies if the database
// had a null or an empty value, but in most cases we don't care about that
// difference. If you need it, use sql.NullString instead.
//
// optionalString should be used when a field is optional. It must be used if there
// is a unique constraint on the optional field.
type optionalString string

func (s *optionalString) Scan(value any) error {
	if value == nil {
		return nil
	}

	var ns sql.NullString
	err := ns.Scan(value)
	*s = (optionalString)(ns.String)
	return err
}

func (s optionalString) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}
