package migrator

import (
	"errors"
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
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Person{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("people")
		},
	},
	{
		ID: "201608301430",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Pet{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("pets")
		},
	},
}

var extendedMigrations = append(migrations, &Migration{
	ID: "201807221927",
	Migrate: func(tx *gorm.DB) error {
		return tx.AutoMigrate(&Book{})
	},
	Rollback: func(tx *gorm.DB) error {
		return tx.Migrator().DropTable("books")
	},
})

var failingMigration = []*Migration{
	{
		ID: "201904231300",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Book{}); err != nil {
				return err
			}
			return errors.New("this transaction should be rolled back")
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	},
}

type Person struct {
	gorm.Model
	Name string
}

type Pet struct {
	gorm.Model
	Name     string
	PersonID int
}

type Book struct {
	gorm.Model
	Name     string
	PersonID int
}

func TestMigration(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, migrations)

		err := m.Migrate()
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(2), tableCount(t, db, "migrations"))

		err = m.RollbackTo(migrations[len(migrations)-2].ID)
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(1), tableCount(t, db, "migrations"))

		err = m.RollbackTo(initSchemaMigrationID)
		assert.NilError(t, err)
		assert.Assert(t, !db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(0), tableCount(t, db, "migrations"))
	})
}

func TestRollbackTo(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, extendedMigrations)

		// First, apply all migrations.
		err := m.Migrate()
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, db.Migrator().HasTable(&Pet{}))
		assert.Assert(t, db.Migrator().HasTable(&Book{}))
		assert.Equal(t, int64(3), tableCount(t, db, "migrations"))

		// Rollback to the first migration: only the last 2 migrations are expected to be rolled back.
		err = m.RollbackTo("201608301400")
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Assert(t, !db.Migrator().HasTable(&Book{}))
		assert.Equal(t, int64(1), tableCount(t, db, "migrations"))
	})
}

// If initSchema is defined, but no migrations are provided,
// then initSchema is executed.
func TestInitSchemaNoMigrations(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, []*Migration{})
		m.options.InitSchema = func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Person{}); err != nil {
				return err
			}
			if err := tx.AutoMigrate(&Pet{}); err != nil {
				return err
			}
			return nil
		}

		assert.NilError(t, m.Migrate())
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(1), tableCount(t, db, "migrations"))
	})
}

// If initSchema is defined and migrations are provided,
// then initSchema is executed and the migration IDs are stored,
// even though the relevant migrations are not applied.
func TestInitSchemaWithMigrations(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, migrations)
		m.options.InitSchema = func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Person{}); err != nil {
				return err
			}
			return nil
		}

		assert.NilError(t, m.Migrate())
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(3), tableCount(t, db, "migrations"))
	})
}

// If the schema has already been initialised,
// then initSchema() is not executed, even if defined.
func TestInitSchemaAlreadyInitialised(t *testing.T) {
	type Car struct {
		gorm.Model
	}

	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, []*Migration{})

		// Migrate with empty initialisation
		m.options.InitSchema = func(tx *gorm.DB) error {
			return nil
		}
		assert.NilError(t, m.Migrate())

		// Then migrate again, this time with a non-empty initialisation
		// This second initialisation should not happen!
		m.options.InitSchema = func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Car{}); err != nil {
				return err
			}
			return nil
		}
		assert.NilError(t, m.Migrate())

		assert.Assert(t, !db.Migrator().HasTable(&Car{}))
		assert.Equal(t, int64(1), tableCount(t, db, "migrations"))
	})
}

// If the schema has not already been initialised,
// but any other migration has already been applied,
// then initSchema() is not executed, even if defined.
func TestInitSchemaExistingMigrations(t *testing.T) {
	type Car struct {
		gorm.Model
	}

	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, DefaultOptions, migrations)

		// Migrate without initialisation
		assert.NilError(t, m.Migrate())

		// Then migrate again, this time with a non-empty initialisation
		// This initialisation should not happen!
		m.options.InitSchema = func(tx *gorm.DB) error {
			if err := tx.AutoMigrate(&Car{}); err != nil {
				return err
			}
			return nil
		}
		assert.NilError(t, m.Migrate())

		assert.Assert(t, !db.Migrator().HasTable(&Car{}))
		assert.Equal(t, int64(2), tableCount(t, db, "migrations"))
	})
}

func TestMissingID(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		migrationsMissingID := []*Migration{
			{
				Migrate: func(tx *gorm.DB) error {
					return nil
				},
			},
		}

		m := New(db, DefaultOptions, migrationsMissingID)
		assert.ErrorContains(t, m.Migrate(), "migration is missing an ID")
	})
}

func TestReservedID(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		migrationsReservedID := []*Migration{
			{
				ID: "SCHEMA_INIT",
				Migrate: func(tx *gorm.DB) error {
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
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		migrationsDuplicatedID := []*Migration{
			{
				ID: "201705061500",
				Migrate: func(tx *gorm.DB) error {
					return nil
				},
			},
			{
				ID: "201705061500",
				Migrate: func(tx *gorm.DB) error {
					return nil
				},
			},
		}

		m := New(db, DefaultOptions, migrationsDuplicatedID)
		err := m.Migrate()
		assert.ErrorContains(t, err, "duplicate migration ID: 201705061500")
	})
}

func TestEmptyMigrationList(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		t.Run("with empty list", func(t *testing.T) {
			m := New(db, DefaultOptions, []*Migration{})
			err := m.Migrate()
			assert.Error(t, err, "there are no migrations")
		})

		t.Run("with nil list", func(t *testing.T) {
			m := New(db, DefaultOptions, nil)
			err := m.Migrate()
			assert.Error(t, err, "there are no migrations")
		})
	})
}

func TestMigration_WithUseTransactions(t *testing.T) {
	options := DefaultOptions
	options.UseTransaction = true

	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		m := New(db, options, migrations)

		err := m.Migrate()
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(2), tableCount(t, db, "migrations"))

		err = m.RollbackTo(migrations[len(migrations)-2].ID)
		assert.NilError(t, err)
		assert.Assert(t, db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(1), tableCount(t, db, "migrations"))

		err = m.RollbackTo(initSchemaMigrationID)
		assert.NilError(t, err)
		assert.Assert(t, !db.Migrator().HasTable(&Person{}))
		assert.Assert(t, !db.Migrator().HasTable(&Pet{}))
		assert.Equal(t, int64(0), tableCount(t, db, "migrations"))
	}, "postgres", "sqlite3", "mssql")
}

func TestMigration_WithUseTransactionsShouldRollback(t *testing.T) {
	options := DefaultOptions
	options.UseTransaction = true

	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		assert.Assert(t, true)
		m := New(db, options, failingMigration)

		// Migration should return an error and not leave around a Book table
		err := m.Migrate()
		assert.Error(t, err, "this transaction should be rolled back")
		assert.Assert(t, !db.Migrator().HasTable(&Book{}))
	}, "postgres", "sqlite3", "mssql")
}

func TestMigrate_WithUnknownMigrationsInTable(t *testing.T) {
	forEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		options := DefaultOptions
		m := New(db, options, migrations)

		// Migrate without initialisation
		assert.NilError(t, m.Migrate())

		n := New(db, DefaultOptions, migrations[:1])
		assert.NilError(t, n.Migrate())
	})
}

func tableCount(t *testing.T, db *gorm.DB, tableName string) (count int64) {
	assert.NilError(t, db.Table(tableName).Count(&count).Error)
	return
}

func forEachDatabase(t *testing.T, fn func(t *testing.T, database *gorm.DB), dialects ...string) {
	dir := t.TempDir()

	databases := []database{
		{dialect: "sqlite3", driver: sqlite.Open("file:" + filepath.Join(dir, "sqlite3.db"))},
	}

	if pg := os.Getenv("PG_CONN_STRING"); pg != "" {
		databases = append(databases, database{
			dialect: "postgres", driver: postgres.Open(pg),
		})
	}

	for _, database := range databases {
		if len(dialects) > 0 && !contains(dialects, database.dialect) {
			t.Skipf("test is not supported by [%s] dialect", database.dialect)
		}

		// Ensure defers are not stacked up for each DB
		t.Run(database.driver.Name(), func(t *testing.T) {
			db, err := gorm.Open(database.driver, &gorm.Config{})
			assert.NilError(t, err, "Could not connect to database %s, %v", database.dialect, err)

			// ensure tables do not exists
			assert.NilError(t, db.Migrator().DropTable("migrations", "people", "pets"))

			fn(t, db)
		})
	}
}

func contains(haystack []string, needle string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}
