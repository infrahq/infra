package migrator

import (
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
)

type database struct {
	dialect string
	driver  gorm.Dialector
}

var migrations = []*Migration{
	{
		ID: "201608301400",
		Migrate: func(tx DB) error {
			_, err := tx.Exec(Person{}.Schema())
			return err
		},
		Rollback: func(tx DB) error {
			_, err := tx.Exec(`DROP TABLE IF EXISTS people`)
			return err
		},
	},
	{
		ID: "201608301430",
		Migrate: func(tx DB) error {
			_, err := tx.Exec(Pet{}.Schema())
			return err
		},
		Rollback: func(tx DB) error {
			_, err := tx.Exec(`DROP TABLE IF EXISTS pets`)
			return err
		},
	},
}

var extendedMigrations = append(migrations, &Migration{
	ID: "201807221927",
	Migrate: func(tx DB) error {
		_, err := tx.Exec(Book{}.Schema())
		return err
	},
	Rollback: func(tx DB) error {
		_, err := tx.Exec(`DROP TABLE IF EXISTS books`)
		return err
	},
})

type Person struct {
	gorm.Model
	Name string
}

func (p Person) Schema() string {
	return `
CREATE TABLE people (
	id integer PRIMARY KEY,
	created_at text,
	updated_at text,
	deleted_at text,
	name text
);`
}

type Pet struct {
	gorm.Model
	Name     string
	PersonID int
}

func (p Pet) Schema() string {
	return `
CREATE TABLE pets (
	id integer PRIMARY KEY,
	created_at text,
	updated_at text,
	deleted_at text,
	name text,
	person_id integer
);`
}

type Book struct {
	gorm.Model
	Name     string
	PersonID int
}

func (b Book) Schema() string {
	return `
CREATE TABLE books (
	id integer PRIMARY KEY,
	created_at text,
	updated_at text,
	deleted_at text,
	name text,
	person_id integer
);`
}

func TestMigration_RunsNewMigrations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		initEmptyMigrations(t, db)
		m := New(db, DefaultOptions, migrations)

		err := m.Migrate()
		assert.NilError(t, err)
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, HasTable(db, "pets"))
		expected := []string{initSchemaMigrationID, "201608301400", "201608301430"}
		assert.DeepEqual(t, migrationIDs(t, db), expected)

		err = m.RollbackTo(migrations[len(migrations)-2].ID)
		assert.NilError(t, err)
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, !HasTable(db, "pets"))
		expected = []string{initSchemaMigrationID, "201608301400"}
		assert.DeepEqual(t, migrationIDs(t, db), expected)

		err = m.RollbackTo(initSchemaMigrationID)
		assert.NilError(t, err)
		assert.Assert(t, !HasTable(db, "people"))
		assert.Assert(t, !HasTable(db, "pets"))
		expected = []string{initSchemaMigrationID}
		assert.DeepEqual(t, migrationIDs(t, db), expected)
	})
}

func initEmptyMigrations(t *testing.T, db DB) {
	t.Helper()
	m := New(db, DefaultOptions, nil)
	err := m.Migrate()
	assert.NilError(t, err)
}

func migrationIDs(t *testing.T, db DB) []string {
	var ids []string
	rows, err := db.Query(`SELECT id from migrations`)
	assert.NilError(t, err)

	for rows.Next() {
		var id string
		assert.NilError(t, rows.Scan(&id))
		ids = append(ids, id)
	}

	assert.NilError(t, rows.Close())
	return ids
}

func TestRollbackTo(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		initEmptyMigrations(t, db)
		m := New(db, DefaultOptions, extendedMigrations)

		// First, apply all migrations.
		err := m.Migrate()
		assert.NilError(t, err)
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, HasTable(db, "pets"))
		assert.Assert(t, HasTable(db, "books"))
		expected := []string{initSchemaMigrationID, "201608301400", "201608301430", "201807221927"}
		assert.DeepEqual(t, migrationIDs(t, db), expected)

		// Rollback to the first migration: only the last 2 migrations are expected to be rolled back.
		err = m.RollbackTo("201608301400")
		assert.NilError(t, err)
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, !HasTable(db, "pets"))
		assert.Assert(t, !HasTable(db, "books"))
		expected = []string{initSchemaMigrationID, "201608301400"}
		assert.DeepEqual(t, migrationIDs(t, db), expected)
	})
}

func TestInitSchemaNoMigrations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		m := New(db, DefaultOptions, []*Migration{})
		m.options.InitSchema = func(tx DB) error {
			if _, err := tx.Exec(Person{}.Schema()); err != nil {
				return err
			}
			if _, err := tx.Exec(Pet{}.Schema()); err != nil {
				return err
			}
			return nil
		}

		assert.NilError(t, m.Migrate())
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, HasTable(db, "pets"))
		assert.Equal(t, int64(1), migrationCount(t, db))
	})
}

// If initSchema is defined and migrations are provided,
// then initSchema is executed and the migration IDs are stored,
// even though the relevant migrations are not applied.
func TestInitSchemaWithMigrations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		m := New(db, DefaultOptions, migrations)
		m.options.InitSchema = func(tx DB) error {
			if _, err := tx.Exec(Person{}.Schema()); err != nil {
				return err
			}
			return nil
		}

		assert.NilError(t, m.Migrate())
		assert.Assert(t, HasTable(db, "people"))
		assert.Assert(t, !HasTable(db, "pets"))
		assert.Equal(t, int64(3), migrationCount(t, db))
	})
}

type Car struct {
	gorm.Model
}

func (c Car) Schema() string {
	return `
CREATE TABLE cars (
	id integer PRIMARY KEY,
	created_at text,
	updated_at text,
	deleted_at text
);`
}

// If the schema has already been initialised,
// then initSchema() is not executed, even if defined.
func TestInitSchemaAlreadyInitialised(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		m := New(db, DefaultOptions, []*Migration{})

		// Migrate with empty initialisation
		m.options.InitSchema = func(tx DB) error {
			return nil
		}
		assert.NilError(t, m.Migrate())

		// Then migrate again, this time with a non-empty initialisation
		// This second initialisation should not happen!
		m.options.InitSchema = func(tx DB) error {
			_, err := tx.Exec(Car{}.Schema())
			return err
		}
		assert.NilError(t, m.Migrate())

		assert.Assert(t, !HasTable(db, "cars"))
		assert.Equal(t, int64(1), migrationCount(t, db))
	})
}

// If the schema has not already been initialised,
// but any other migration has already been applied,
// then initSchema() is not executed, even if defined.
func TestInitSchemaExistingMigrations(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		m := New(db, DefaultOptions, migrations)

		// Migrate without initialisation
		assert.NilError(t, m.Migrate())

		// Then migrate again, this time with a non-empty initialisation
		// This initialisation should not happen!
		m.options.InitSchema = func(tx DB) error {
			_, err := tx.Exec(Car{}.Schema())
			return err
		}
		assert.NilError(t, m.Migrate())

		assert.Assert(t, !HasTable(db, "cars"))
		expected := []string{initSchemaMigrationID, "201608301400", "201608301430"}
		assert.DeepEqual(t, migrationIDs(t, db), expected)
	})
}

func TestMissingID(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		migrationsMissingID := []*Migration{
			{
				Migrate: func(tx DB) error {
					return nil
				},
			},
		}

		m := New(db, DefaultOptions, migrationsMissingID)
		assert.ErrorContains(t, m.Migrate(), "migration is missing an ID")
	})
}

func TestReservedID(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		migrationsReservedID := []*Migration{
			{
				ID: "SCHEMA_INIT",
				Migrate: func(tx DB) error {
					return nil
				},
			},
		}

		m := New(db, DefaultOptions, migrationsReservedID)
		err := m.Migrate()
		assert.ErrorContains(t, err, "migration can not use reserved ID")
	})
}

func TestDuplicatedID(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		migrationsDuplicatedID := []*Migration{
			{
				ID: "201705061500",
				Migrate: func(tx DB) error {
					return nil
				},
			},
			{
				ID: "201705061500",
				Migrate: func(tx DB) error {
					return nil
				},
			},
		}

		m := New(db, DefaultOptions, migrationsDuplicatedID)
		err := m.Migrate()
		assert.ErrorContains(t, err, "duplicate migration ID: 201705061500")
	})
}

func TestMigrate_WithUnknownMigrationsInTable(t *testing.T) {
	runDBTests(t, func(t *testing.T, db DB) {
		options := DefaultOptions
		m := New(db, options, migrations)

		// Migrate without initialisation
		assert.NilError(t, m.Migrate())

		n := New(db, DefaultOptions, migrations[:1])
		assert.NilError(t, n.Migrate())
	})
}

func migrationCount(t *testing.T, db DB) (count int64) {
	t.Helper()
	err := db.QueryRow(`SELECT count(id) from migrations`).Scan(&count)
	assert.NilError(t, err)
	return count
}

func runDBTests(t *testing.T, fn func(t *testing.T, db DB)) {
	dir := t.TempDir()

	databases := []database{
		{dialect: "sqlite3", driver: sqlite.Open("file:" + filepath.Join(dir, "sqlite3.db"))},
	}

	if pg := os.Getenv("POSTGRESQL_CONNECTION"); pg != "" {
		databases = append(databases, database{
			dialect: "postgres", driver: postgres.Open(pg),
		})
	}

	for _, database := range databases {
		// Ensure defers are not stacked up for each DB
		t.Run(database.driver.Name(), func(t *testing.T) {
			db, err := gorm.Open(database.driver, &gorm.Config{})
			assert.NilError(t, err, "Could not connect to database %s, %v", database.dialect, err)

			for _, table := range []string{"migrations", "people", "pets", "books"} {
				err := db.Exec(`DROP TABLE IF EXISTS ` + table).Error
				assert.NilError(t, err)
			}

			fn(t, gormDBShim{DB: db})
		})
	}
}

type gormDBShim struct {
	*gorm.DB
}

func (d gormDBShim) DriverName() string {
	return d.Dialector.Name()
}

func (d gormDBShim) Exec(query string, args ...any) (sql.Result, error) {
	db := d.DB.Exec(query, args...)
	return driver.RowsAffected(db.RowsAffected), db.Error
}

func (d gormDBShim) Query(query string, args ...any) (*sql.Rows, error) {
	return d.DB.Raw(query, args...).Rows()
}

func (d gormDBShim) QueryRow(query string, args ...any) *sql.Row {
	return d.DB.Raw(query, args...).Row()
}

// DefaultOptions used for tests
var DefaultOptions = Options{
	InitSchema: func(tx DB) error {
		return nil
	},
}
