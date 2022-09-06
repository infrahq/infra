package data

import (
	"database/sql"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/uid"
)

type example struct {
	ID    uid.ID
	First string
	Age   int
}

func (e example) Primary() uid.ID {
	return e.ID
}

func (e example) Table() string {
	return "examples"
}

func (e example) Columns() []string {
	return []string{"id", "first", "age"}
}

func (e example) Values() []any {
	return []any{e.ID, e.First, e.Age}
}

func (e example) OnInsert() error {
	return nil
}

func (e example) OnUpdate() error {
	return nil
}

func TestInsert(t *testing.T) {
	e := example{ID: 123, First: "first", Age: 111}
	tx := &txnCapture{}
	err := insert(tx, e)
	assert.NilError(t, err)
	expected := `INSERT INTO examples ( id, first, age ) VALUES ( ?, ?, ? ); `
	assert.Equal(t, tx.query, expected)
	expectedArgs := []any{uid.ID(123), "first", 111}
	assert.DeepEqual(t, tx.args, expectedArgs)
}

type txnCapture struct {
	ReadTxn
	query string
	args  []any
}

func (t *txnCapture) Exec(query string, args ...any) (sql.Result, error) {
	t.query = query
	t.args = args
	return nil, nil
}

func TestUpdate(t *testing.T) {
	e := example{ID: 123, First: "first", Age: 111}
	tx := &txnCapture{}
	err := update(tx, e)
	assert.NilError(t, err)
	expected := `UPDATE examples SET id = ?, first = ?, age = ? WHERE deleted_at is null AND id = ?; `
	assert.Equal(t, tx.query, expected)
	expectedArgs := []any{uid.ID(123), "first", 111, uid.ID(123)}
	assert.DeepEqual(t, tx.args, expectedArgs)
}
