package database

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/generate"
)

type TestingT interface {
	assert.TestingT
	Cleanup(func())
	Fatal(...any)
	Skip(...any)
	Helper()
}

var isEnvironmentCI = os.Getenv("CI") != ""

// PostgresDriver returns a driver for connecting to postgres based on the
// POSTGRESQL_CONNECTION environment variable. The value should be a postgres
// connection string, see
// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING.
//
// schemaSuffix is used to create a schema name to isolate the database from
// other tests. Most tests should specify a schemaSuffix to identify the package
// using the database. Database migration tests will use an empty string for the
// suffix because those tests required a schema with the name "testing".
func PostgresDriver(t TestingT, schemaSuffix string) *Driver {
	t.Helper()
	pgConn, ok := os.LookupEnv("POSTGRESQL_CONNECTION")
	switch {
	case !ok && isEnvironmentCI:
		t.Fatal("CI must test all drivers, set POSTGRESQL_CONNECTION")
	case !ok:
		t.Skip("Set POSTGRESQL_CONNECTION to test against postgresql")
	}

	suffix := strings.NewReplacer("--", "", ";", "", "/", "").Replace(schemaSuffix)
	name := "testing"
	if schemaSuffix != "" {
		name = fmt.Sprintf("testing_%v_%v", suffix, generate.MathRandom(5, generate.CharsetNumbers))
	}
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
