package data

import "database/sql"

// optionalString has the behaviour of sql.NullString allowing a null entry
// in a database column to be scanned as the empty value of a string. Instead
// of using sql.NullString as the field, optionalString allows you to wrap
// a regular string field on a struct.
// With optionalString you lose the Valid field which identifies if the database
// had a null or an empty value, but in most cases we don't care about that
// difference. If you need it, use sql.NullString instead.
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
