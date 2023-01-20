package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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
		name = fmt.Sprintf("test_%v_%v", suffix, generate.MathRandom(3, generate.CharsetNumbers))
	}
	if len(schemaSuffix) >= 24 {
		t.Fatal("schema suffix", schemaSuffix, "must be less than 24 characters")
	}

	db, err := sql.Open("pgx", pgConn)
	assert.NilError(t, err, "connect to postgresql")
	t.Cleanup(func() {
		_, err := db.Exec("DROP SCHEMA IF EXISTS " + name + " CASCADE")
		assert.NilError(t, err)
		assert.NilError(t, db.Close())
	})

	// Drop any leftover schema before creating a new one.
	_, err = db.Exec("DROP SCHEMA IF EXISTS " + name + " CASCADE")
	assert.NilError(t, err)
	_, err = db.Exec("CREATE SCHEMA " + name)
	assert.NilError(t, err)

	dsn := pgConn + " search_path=" + name
	return &Driver{DSN: dsn}
}

type Driver struct {
	// DSN is the connection string that can be used to connect to this
	// database.
	DSN string
}
