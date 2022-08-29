package data

import (
	"strings"

	"github.com/infrahq/infra/internal/server/data/querybuilder"
	"github.com/infrahq/infra/uid"
)

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
	// OnInsert is called by insert to initialize values before inserting.
	OnInsert() error
}

type Updatable interface {
	Insertable
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
	return err
}

func columnsForInsert(table Table) string {
	return strings.Join(table.Columns(), ", ")
}

func placeholderForColumns(table Table) string {
	columns := table.Columns()
	result := make([]string, len(columns))
	for i := range columns {
		result[i] = "?"
	}
	return strings.Join(result, ", ")
}

func update(tx WriteTxn, item Updatable) error {
	if err := item.OnUpdate(); err != nil {
		return err
	}
	setOrg(tx, item)

	query := querybuilder.New("UPDATE")
	query.B(item.Table())
	query.B("SET")
	query.B(columnsForUpdate(item), item.Values()...)
	query.B("WHERE deleted_at is null AND id = ?;", item.Primary())
	_, err := tx.Exec(query.String(), query.Args...)
	return err
}

func columnsForUpdate(table Table) string {
	return strings.Join(table.Columns(), " = ?, ") + " = ?"
}

func columnsForSelect(tableAlias string, table Table) string {
	if tableAlias == "" {
		return strings.Join(table.Columns(), ", ")
	}
	return tableAlias + "." + strings.Join(table.Columns(), ", "+tableAlias+".")
}
