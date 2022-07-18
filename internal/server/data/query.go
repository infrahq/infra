package data

import (
	"strings"

	"github.com/infrahq/infra/uid"
)

type queryBuilder struct {
	query strings.Builder
	Args  []interface{}
}

// Query initializes a queryBuilder and returns it populated with selectFrom.
// The selectFrom string can generally contain any SELECT, FROM, or JOIN
// clauses, and may contain the entire query as long as there are no
// query parameters.
// Use queryBuilder.B to add sections of a query with parameters.
func Query(selectFrom string) *queryBuilder {
	q := &queryBuilder{}
	q.query.WriteString(selectFrom + " ")
	return q
}

func (q *queryBuilder) B(clause string, args ...interface{}) {
	q.query.WriteString(clause + " ")
	q.Args = append(q.Args, args...)
}

// String returns the query string, which is used as the first parameter to
// Tx.Exec, or Tx.Query. You must also pass q.Args as the varargs.
func (q *queryBuilder) String() string {
	return q.query.String()
}

// IDOrNameQuery is used to query for a single item by ID or name. ID and name
// are mutually exclusive, only one may be set to a non-zero value.
type IDOrNameQuery struct {
	ID   uid.ID
	Name string
}

// ByIDQ returns an IDOrNameQuery that will query by ID.
// TODO: rename to ByID once all queries are ported to the new interface.
func ByIDQ(id uid.ID) IDOrNameQuery {
	return IDOrNameQuery{ID: id}
}

// ByNameQ returns an IDOrNameQuery that will query by Name.
// TODO: rename to ByName once all queries are ported to the new interface.
func ByNameQ(name string) IDOrNameQuery {
	return IDOrNameQuery{Name: name}
}
