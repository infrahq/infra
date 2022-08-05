package database

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

type TestingT interface {
	assert.TestingT
	Cleanup(func())
}

func PostgresDriver(t TestingT) gorm.Dialector {
	pgConn, ok := os.LookupEnv("POSTGRESQL_CONNECTION")
	if !ok {
		return nil
	}

	db, err := gorm.Open(postgres.Open(pgConn))
	assert.NilError(t, err, "connect to postgresql")
	t.Cleanup(func() {
		assert.NilError(t, db.Exec("DROP SCHEMA IF EXISTS testing CASCADE").Error)
		sqlDB, err := db.DB()
		assert.NilError(t, err)
		assert.NilError(t, sqlDB.Close())
	})
	assert.NilError(t, db.Exec("CREATE SCHEMA testing").Error)

	pgsql := postgres.Open(pgConn + " search_path=testing")
	return pgsql
}
