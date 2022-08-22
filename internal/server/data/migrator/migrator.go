package migrator

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal/logging"
)

const initSchemaMigrationID = "SCHEMA_INIT"

// Options used by the Migrator to perform database migrations.
type Options struct {
	// InitSchema is used to create the database when no migrations table exists.
	// This function should create all tables, and constraints. After this
	// function is run, migrator will create the migrations table and populate
	// it with the IDs of all the currently defined migrations.
	InitSchema func(DB) error

	// LoadKey is an optional function to initialize an encryption key from the
	// database, that is used to encrypt other fields.
	LoadKey func(DB) error
}

// Migration defines a database migration, and an optional rollback.
type Migration struct {
	// ID is the migration identifier. Usually a timestamp like "2016-01-02T15:04".
	ID string
	// Migrate is a function that will br executed while running this migration.
	Migrate func(DB) error
	// Rollback will be executed on rollback. Can be nil.
	Rollback func(DB) error
}

type DB interface {
	// DriverName returns the name of the database driver.
	DriverName() string

	Exec(stmt string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Migrator performs database migrations.
type Migrator struct {
	tx         DB
	options    Options
	migrations []*Migration
}

// New returns a new Migrator.
func New(db DB, options Options, migrations []*Migration) *Migrator {
	if options.LoadKey == nil {
		options.LoadKey = func(tx DB) error {
			return nil
		}
	}
	return &Migrator{
		tx:         db,
		options:    options,
		migrations: migrations,
	}
}

// Migrate runs all the migrations that have not yet been applied to the
// database. Migrate may follow one of three flows:
//
//  1. If the initial schema has not yet been applied then Migrate will run
//     Options.InitSchema, and then exit.
//  2. If all the migrations have already been applied then Migrate will do
//     nothing.
//  3. If there are migrations in the list that have not yet been applied then
//     Migrate will run them in order.
func (g *Migrator) Migrate() error {
	if g.options.InitSchema == nil && len(g.migrations) == 0 {
		return fmt.Errorf("there are no migrations")
	}
	if err := g.validate(); err != nil {
		return err
	}

	if err := g.createMigrationTableIfNotExists(); err != nil {
		return err
	}

	initSchema, err := g.mustInitializeSchema()
	switch {
	case err != nil:
		return err
	case initSchema:
		return g.runInitSchema()
	}

	if err := g.options.LoadKey(g.tx); err != nil {
		return fmt.Errorf("load key: %w", err)
	}

	for _, migration := range g.migrations {
		if err := g.runMigration(migration); err != nil {
			return err
		}
	}
	return nil
}

func (g *Migrator) validate() error {
	lookup := make(map[string]struct{}, len(g.migrations))

	for _, m := range g.migrations {
		switch m.ID {
		case "":
			return fmt.Errorf("migration is missing an ID")
		case initSchemaMigrationID:
			return fmt.Errorf("migration can not use reserved ID: %v", m.ID)
		}
		if _, ok := lookup[m.ID]; ok {
			return fmt.Errorf("duplicate migration ID: %v", m.ID)
		}
		lookup[m.ID] = struct{}{}
	}
	return nil
}

func (g *Migrator) checkIDExist(migrationID string) error {
	if migrationID == initSchemaMigrationID {
		return nil
	}
	for _, migrate := range g.migrations {
		if migrate.ID == migrationID {
			return nil
		}
	}
	return fmt.Errorf("migration ID %v does not exist", migrationID)
}

// RollbackTo undoes migrations up to the given migration that matches the `migrationID`.
// Migration with the matching `migrationID` is not rolled back.
func (g *Migrator) RollbackTo(migrationID string) error {
	if len(g.migrations) == 0 {
		return fmt.Errorf("there are no migrations")
	}

	if err := g.checkIDExist(migrationID); err != nil {
		return err
	}

	for i := len(g.migrations) - 1; i >= 0; i-- {
		migration := g.migrations[i]
		if migration.ID == migrationID {
			break
		}
		switch migrationRan, err := g.migrationRan(migration); {
		case err != nil:
			return err
		case migrationRan:
			if err := g.rollbackMigration(migration); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Migrator) rollbackMigration(m *Migration) error {
	if m.Rollback == nil {
		return errors.New("migration can not be rollback back")
	}

	if err := m.Rollback(g.tx); err != nil {
		return err
	}
	_, err := g.tx.Exec("DELETE FROM migrations WHERE id = ?", m.ID)
	return err
}

func (g *Migrator) runInitSchema() error {
	if err := g.options.InitSchema(g.tx); err != nil {
		return err
	}
	if err := g.insertMigration(initSchemaMigrationID); err != nil {
		return err
	}
	for _, migration := range g.migrations {
		if err := g.insertMigration(migration.ID); err != nil {
			return err
		}
	}
	return g.options.LoadKey(g.tx)
}

func (g *Migrator) runMigration(migration *Migration) error {
	switch migrationRan, err := g.migrationRan(migration); {
	case err != nil:
		return err
	case migrationRan:
		return nil
	}

	logging.Infof("Running migration %s", migration.ID)
	if err := migration.Migrate(g.tx); err != nil {
		return fmt.Errorf("failed to apply migration %v: %w", migration.ID, err)
	}
	return g.insertMigration(migration.ID)
}

func (g *Migrator) createMigrationTableIfNotExists() error {
	// TODO: replace gorm helper
	if HasTable(g.tx, "migrations") {
		return nil
	}

	_, err := g.tx.Exec("CREATE TABLE migrations (id VARCHAR(255) PRIMARY KEY)")
	return err
}

// TODO: select all values from the table once, instead of selecting each
// individually
func (g *Migrator) migrationRan(m *Migration) (bool, error) {
	var count int64
	err := g.tx.QueryRow(`SELECT count(id) FROM migrations WHERE id = ?`, m.ID).Scan(&count)
	return count > 0, err
}

func (g *Migrator) mustInitializeSchema() (bool, error) {
	migrationRan, err := g.migrationRan(&Migration{ID: initSchemaMigrationID})
	if err != nil {
		return false, err
	}
	if migrationRan {
		return false, nil
	}

	// If the ID doesn't exist, we also want the list of migrations to be empty
	var count int64
	err = g.tx.QueryRow(`SELECT count(id) FROM migrations`).Scan(&count)
	return count == 0, err
}

func (g *Migrator) insertMigration(id string) error {
	_, err := g.tx.Exec("INSERT INTO migrations (id) VALUES (?)", id)
	return err
}
