package database

import (
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

type TestingT interface {
	assert.TestingT
	Cleanup(func())
	Helper()
}

// PostgresDriver returns a driver for connecting to postgres based on the
// POSTGRESQL_CONNECTION environment variable. The value should be a postgres
// connection string, see
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING.
//
// schemaSuffix is used to create a schema name to isolate the database from
// other tests.
func PostgresDriver(t TestingT, schemaSuffix string) *Driver {
	t.Helper()
	pgConn, ok := os.LookupEnv("POSTGRESQL_CONNECTION")
	if !ok {
		return nil
	}

	suffix := strings.NewReplacer("--", "", ";", "", "/", "").Replace(schemaSuffix)
	name := "testing" + suffix
	db, err := gorm.Open(postgres.Open(pgConn))
	assert.NilError(t, err, "connect to postgresql")
	t.Cleanup(func() {
		assert.NilError(t, db.Exec("DROP SCHEMA IF EXISTS "+name+" CASCADE").Error)
		sqlDB, err := db.DB()
		assert.NilError(t, err)
		assert.NilError(t, sqlDB.Close())
	})

	// Drop any leftover schema before creating a new one.
	assert.NilError(t, db.Exec("DROP SCHEMA IF EXISTS "+name+" CASCADE").Error)
	assert.NilError(t, db.Exec("CREATE SCHEMA "+name).Error)

	dsn := pgConn + " search_path=" + name
	pgsql := postgres.Open(dsn)
	return &Driver{Dialector: pgsql, DSN: dsn}
}

type Driver struct {
	Dialector gorm.Dialector
	// DSN is the connection string that can be used to connect to this
	// database.
	DSN string
}
