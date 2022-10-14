package migrator

import (
	"strings"

	"github.com/infrahq/infra/internal/logging"
)

// HasTable returns true if the database has a table with name. Returns
// false if the table does not exist, or if there was a failure querying the
// database.
func HasTable(tx DB, name string) bool {
	var count int
	stmt := `
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = CURRENT_SCHEMA()
		AND table_name = ? AND table_type = 'BASE TABLE'
	`
	if err := tx.QueryRow(stmt, name).Scan(&count); err != nil {
		logging.L.Warn().Err(err).Msg("failed to check if table exists")
		return false
	}
	return count != 0
}

// HasColumn returns true if the database table has the column. Returns false if
// the database table does not have the column, or if there was a failure querying
// the database.
func HasColumn(tx DB, table string, column string) bool {
	var count int

	stmt := `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA()
		AND table_name = ? AND column_name = ?
	`
	if err := tx.QueryRow(stmt, table, column).Scan(&count); err != nil {
		logging.L.Warn().Err(err).Msg("failed to check if column exists")
		return false
	}
	return count != 0
}

// HasConstraint returns true if the database table has the constraint. Returns
// false if the database table does not have the constraint, or if there was a
// failure querying the database.
func HasConstraint(tx DB, table string, constraint string) bool {
	var count int
	stmt := `
		SELECT count(*)
		FROM information_schema.table_constraints
		WHERE table_schema = CURRENT_SCHEMA()
		AND table_name = ? AND constraint_name = ?
	`
	if err := tx.QueryRow(stmt, table, constraint).Scan(&count); err != nil {
		logging.L.Warn().Err(err).Msg("failed to check if constraint exists")
		return false
	}
	return count != 0
}

// HasFunction returns true if the database already has the function. Returns
// false if the database table does not have the function, or if there was a
// failure querying the database.
func HasFunction(tx DB, funcName string) bool {
	stmt := `
		SELECT count(*)
		FROM pg_proc
		INNER JOIN pg_namespace ON pg_proc.pronamespace = pg_namespace.oid
		WHERE proname = ? AND nspname = CURRENT_SCHEMA()`

	var count int
	// function names are stored in lowercase, so convert to lowercase
	if err := tx.QueryRow(stmt, strings.ToLower(funcName)).Scan(&count); err != nil {
		logging.L.Warn().Err(err).Msg("failed to check if function exists")
		return false
	}
	return count != 0
}
