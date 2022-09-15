/*
Package querybuilder is a small package which provides a single type for building
sql queries. Generally a single type would be too small for a whole package. In
this case a separate packages is used because it is easier to perform static
analysis for potential SQL injection. The query field of Query can only be
directly accessed from this package, so internal/tools/querylinter only needs
to check for use of exported methods.
*/
package querybuilder

import "strings"

// Query builds a sql statement from one or more trusted strings, and a list
// of untrusted arguments.
type Query struct {
	// query is used as the trusted and unescaped query string. Only trusted
	// string literals should be added to query. internal/tools/querylinter
	// provides a linter that checks the use of exported methods, but there is
	// no linter for this package. When making changes to Query ensure
	// that the linter is updated to cover any new methods that may add strings
	// to the query field.
	query strings.Builder
	// Args is a list of untrusted arguments that will be escaped by the
	// database driver when constructing the query statement.
	Args []interface{}
}

// New initializes a Query and returns it populated with stmt.
// The stmt string must be a string literal, that will generally contain
// SELECT, UPDATE, DELETE FROM, or INSERT INTO.
//
// Use Query.B to add sections of a query with parameters.
func New(stmt string) *Query {
	q := &Query{}
	q.query.WriteString(stmt + " ")
	return q
}

// B adds clause and args to the query. The clause must be a trusted string
// literal. Any arguments must be passed as args so that they are properly
// escaped by the database driver.
func (q *Query) B(clause string, args ...interface{}) {
	q.query.WriteString(clause + " ")
	q.Args = append(q.Args, args...)
}

// String returns the query statement, which is used as the first parameter to
// WriteTxn.Exec, or ReadTxn.Query. You must also pass q.Args as the varargs to
// the transaction method.
func (q *Query) String() string {
	return q.query.String()
}
