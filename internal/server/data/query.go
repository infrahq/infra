package data

import (
	"strings"

	"gorm.io/gorm"

	"github.com/infrahq/infra/uid"
)

type queryBuilder struct {
	query strings.Builder
	Args  []interface{}
}

// Query initializes a queryBuilder and returns it populated with stmt.
// The stmt string can generally contain any SELECT, FROM, or JOIN
// clauses, and may contain the entire query as long as there are no
// query parameters.
// Use queryBuilder.B to add sections of a query with parameters.
func Query(stmt string) *queryBuilder {
	q := &queryBuilder{}
	q.query.WriteString(stmt + " ")
	return q
}

// B adds clause and args to the query. The clause must be a trusted string
// literal. Any arguments must be passed as args so that they are properly
// escaped by the database driver.
func (q *queryBuilder) B(clause string, args ...interface{}) {
	q.query.WriteString(clause + " ")
	q.Args = append(q.Args, args...)
}

// String returns the query string, which is used as the first parameter to
// WriteTxn.Exec, or ReadTxn.Query. You must also pass q.Args as the varargs.
func (q *queryBuilder) String() string {
	return q.query.String()
}

type WriteTxn interface {
	Exec(query string, args ...any) *gorm.DB
}

type ReadTxn interface {
	Raw(query string, args ...any) *gorm.DB
}

type Table interface {
	Table() string
	// Columns returns the names of the tables columns.
	Columns() []string
}

type Insertable interface {
	Table
	// Values returns the values for all fields. The values must be in the same
	// order as the column names returned by Columns.
	Values() []any
}

type Updatable interface {
	Insertable
	// Primary returns the value for the field that is mapped to the primary key
	// of the table.
	Primary() uid.ID
}

type Deletable interface {
	Table() string
	Primary() uid.ID
}

type Selectable interface {
	Table
	// ScanFields returns pointers to all the fields, which should be used in
	// sql.Rows.Scan. The fields must be in the same order as the column names
	// returned by Columns.
	ScanFields() []any
}

func insert(tx WriteTxn, item Insertable) error {
	query := Query("INSERT INTO")
	query.B(item.Table())
	query.B("(")
	query.B(columnsForInsert(item.Columns()))
	query.B(") VALUES (")
	query.B(placeholderForColumns(item.Columns()), item.Values()...)
	query.B(");")
	err := tx.Exec(query.String(), query.Args...).Error
	return err
}

func columnsForInsert(columns []string) string {
	return strings.Join(columns, ", ")
}

func placeholderForColumns(columns []string) string {
	result := make([]string, len(columns))
	for i := range columns {
		result[i] = "?"
	}
	return strings.Join(result, ", ")
}

func update(tx WriteTxn, item Updatable) error {
	query := Query("UPDATE")
	query.B(item.Table())
	query.B("SET")
	query.B(columnsForUpdate(item.Columns()), item.Values()...)
	query.B("WHERE id = ?;", item.Primary())
	err := tx.Exec(query.String(), query.Args...).Error
	return err
}

func columnsForUpdate(columns []string) string {
	return strings.Join(columns, " = ?, ") + " = ?"
}
