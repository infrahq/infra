package migrator

import "gorm.io/gorm"

type Tx interface {
	Exec(stmt string, args ...any) *gorm.DB
}

// HasTable returns true if the database has a table with name. Returns
// false if the table does not exist, or if there was a failure querying the
// database.
func HasTable(tx *gorm.DB, name string) bool {
	var count int
	stmt := `
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = CURRENT_SCHEMA()
		AND table_name = ? AND table_type = 'BASE TABLE'
	`
	if tx.Dialector.Name() == "sqlite" {
		stmt = `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`
	}

	if err := tx.Raw(stmt, name).Scan(&count).Error; err != nil {
		return false
	}
	return count != 0
}

// HasColumn returns true if the database table has the column. Returns false if
// the database table does not have the column, or if there was a failure querying
// the database.
func HasColumn(tx *gorm.DB, table string, column string) bool {
	var count int

	stmt := `
		SELECT count(*)
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA()
		AND table_name = ? AND column_name = ?
	`

	if tx.Dialector.Name() == "sqlite" {
		stmt = `
			SELECT count(*)
			FROM sqlite_master
			WHERE type = 'table' AND name = ?
			AND sql LIKE ?
		`
		column = "%`" + column + "`%"
	}

	if err := tx.Raw(stmt, table, column).Scan(&count).Error; err != nil {
		return false
	}
	return count != 0
}
