package migrator

import "gorm.io/gorm"

// HasTable returns true if the database already has a table with name. Returns
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
		stmt = `SELECT count(*) FROM sqlite_master WHERE type = 'table' and name = ?`
	}

	if err := tx.Raw(stmt, name).Scan(&count).Error; err != nil {
		return false
	}
	return count != 0
}
