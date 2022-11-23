package data

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/uid"
)

// ReadTxn can perform read queries and contains metadata about the request.
type ReadTxn interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	OrganizationID() uid.ID
}

// WriteTxn extends ReadTxn by adding write queries.
type WriteTxn interface {
	ReadTxn
	Exec(sql string, values ...interface{}) (sql.Result, error)
}

type Table interface {
	// Table returns the name of the database table.
	Table() string
	// Columns returns the names of the table's columns. Columns must return
	// a slice literal where every item in the slice is a string literal.
	// The value returned by Columns is used as a trusted string in queries and
	// will not be escaped. internal/tools/querylinter will vet all
	// implementations of this method to ensure that only string literals are
	// returned.
	// If the definition of this method changes then internal/tools/querylinter
	// must be updated accordingly.
	Columns() []string
}

type Insertable interface {
	Table
	// Values returns the values for all fields. The values must be in the same
	// order as the column names returned by Columns.
	Values() []any
	// OnInsert is called by insert to initialize values before inserting.
	OnInsert() error
}

type Updatable interface {
	Table
	// Values returns the values for all fields. The values must be in the same
	// order as the column names returned by Columns.
	Values() []any
	// Primary returns the value for the field that is mapped to the primary key
	// of the table.
	Primary() uid.ID
	// OnUpdate is called by update to initialize values before updating.
	OnUpdate() error
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

// insert an item into the database using tx. insert is a convenience function
// for the common case. Not all create functions use this function. Special
// cases warrant copying the implementation of insert to make the necessary changes.
func insert(tx WriteTxn, item Insertable) error {
	if err := item.OnInsert(); err != nil {
		return err
	}
	setOrg(tx, item)

	query := querybuilder.New("INSERT INTO")
	query.B(item.Table())
	query.B("(")
	query.B(columnsForInsert(item))
	query.B(") VALUES (")
	query.B(placeholderForColumns(item), item.Values()...)
	query.B(");")
	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

// columnsForInsert returns the list of columns names for table as a string
// appropriate for an INSERT statement.
//
//	column1, column2, column3, ...
//
// columnsForInsert is a special function that is not checked by
// internal/tools/querylinter. If the arguments to this function change
// the linter will likely need to be updated.
// The return value must only include trusted strings from the source code,
// never untrusted user input.
func columnsForInsert(table Table) string {
	return strings.Join(table.Columns(), ", ")
}

// placeholderForColumns returns a list of argument placeholders as a string
// appropriate for an INSERT statement. The number of placeholders is equal to
// the number of columns on table.
//
// placeholderForColumns is a special function that is not checked by
// internal/tools/querylinter. If the arguments to this function change
// the linter will likely need to be updated.
// The return value must only include trusted strings from the source code,
// never untrusted user input.
func placeholderForColumns(table Table) string {
	columns := table.Columns()
	result := make([]string, len(columns))
	for i := range columns {
		result[i] = "?"
	}
	return strings.Join(result, ", ")
}

// update an item in the database using tx. update is a convenience function
// for the common case. Not all UpdateType functions use this function. Special
// cases warrant copying the implementation of update to make the necessary changes.
func update(tx WriteTxn, item Updatable) error {
	if err := item.OnUpdate(); err != nil {
		return err
	}
	setOrg(tx, item)

	query := querybuilder.New("UPDATE")
	query.B(item.Table())
	query.B("SET")
	query.B(columnsForUpdate(item), item.Values()...)
	query.B("WHERE deleted_at is null")
	query.B("AND id = ?", item.Primary())
	if isOrgMember(item) {
		query.B("AND organization_id = ?;", tx.OrganizationID())
	}
	_, err := tx.Exec(query.String(), query.Args...)
	return handleError(err)
}

// columnsForUpdate returns a list of column assignment expressions as a string
// appropriate for an UPDATE statement.
//
//	column1 = ?, column2 = ?, column3 = ?, ...
//
// columnsForUpdate is a special function that is not checked by
// internal/tools/querylinter. If the arguments to this function change
// the linter will likely need to be updated.
// The return value must only include trusted strings from the source code,
// never untrusted user input.
func columnsForUpdate(table Table) string {
	return strings.Join(table.Columns(), " = ?, ") + " = ?"
}

// columnsForSelect returns a list of column names as a string appropriate for
// a SELECT statement.
//
//	table.column1, table.column2, table.column3, ...
//
// columnsForSelect is a special function that is not checked by
// internal/tools/querylinter. If the arguments to this function change
// the linter will likely need to be updated.
// The return value must only include trusted strings from the source code,
// never untrusted user input.
func columnsForSelect(table Table) string {
	name := table.Table()
	return name + "." + strings.Join(table.Columns(), ", "+name+".")
}

// scanRows iterates over rows and builds a slice of T by scanning each row
// into fields. rows is closed before returning.
func scanRows[T any](rows *sql.Rows, fields func(*T) []any) ([]T, error) {
	defer rows.Close()

	var result []T
	for rows.Next() {
		var target T

		if err := rows.Scan(fields(&target)...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, target)
	}
	return result, rows.Err()
}

// countRows performs a query that returns the number of rows in the table where
// deleted_at is null. The count includes all organizations.
//
// To get counts for tables that do not have a deleted_at column, or to scope
// the count, copy the implementation of this function and add the necessary
// parameters to the query.
func countRows(tx ReadTxn, table Table) (int64, error) {
	query := querybuilder.New("SELECT count(*) FROM")
	query.B(table.Table())
	query.B("WHERE deleted_at is null")

	var count int64
	err := tx.QueryRow(query.String(), query.Args...).Scan(&count)
	return count, handleError(err)
}

// queryInClause adds a (?, ?, ?, ...) string to the query string, and all the
// items to the query.Args. An empty slice will add (null) to the query string
// which will match no rows. This is done to prevent a syntax error.
// queryInClause should generally be used after an IN or NOT IN condition in the
// query.
func queryInClause[T any](query *querybuilder.Query, items []T) {
	if len(items) == 0 {
		query.B("(null)")
		return
	}
	query.B("(")
	for i, item := range items {
		if i != 0 {
			query.B(",")
		}
		query.B("?", item)
	}
	query.B(")")
}
