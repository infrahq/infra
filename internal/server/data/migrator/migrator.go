package migrator

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
)

const initSchemaMigrationID = "SCHEMA_INIT"

// Options define options for all migrations.
type Options struct {
	// UseTransaction makes Migrator execute migrations inside a single transaction.
	// Keep in mind that not all databases support DDL commands inside transactions.
	UseTransaction bool

	// InitSchema is used to create the database when no migrations table exists.
	// This function should create all tables, and constraints. After this
	// function is run, migrator will create the migrations table and populate
	// it with the IDs of all the currently defined migrations.
	InitSchema func(*gorm.DB) error
}

// Migration represents a database migration (a modification to be made on the database).
type Migration struct {
	// ID is the migration identifier. Usually a timestamp like "201601021504".
	ID string
	// Migrate is a function that will br executed while running this migration.
	Migrate func(*gorm.DB) error
	// Rollback will be executed on rollback. Can be nil.
	Rollback func(*gorm.DB) error
}

// Migrator represents a collection of all migrations of a database schema.
type Migrator struct {
	db         *gorm.DB
	tx         *gorm.DB
	options    Options
	migrations []*Migration
}

// DefaultOptions can be used if you don't want to think about options.
var DefaultOptions = Options{
	UseTransaction: false,
	InitSchema: func(db *gorm.DB) error {
		return nil
	},
}

// New returns a new Migrator.
func New(db *gorm.DB, options Options, migrations []*Migration) *Migrator {
	return &Migrator{
		db:         db,
		options:    options,
		migrations: migrations,
	}
}

// Migrate executes all migrations that did not run yet.
func (g *Migrator) Migrate() error {
	if g.options.InitSchema == nil && len(g.migrations) == 0 {
		return fmt.Errorf("there are no migrations")
	}

	if err := g.validate(); err != nil {
		return err
	}

	rollback := g.begin()
	defer rollback()

	if err := g.createMigrationTableIfNotExists(); err != nil {
		return err
	}

	canInitializeSchema, err := g.shouldInitializeSchema()
	if err != nil {
		return err
	}
	if canInitializeSchema {
		if err := g.runInitSchema(); err != nil {
			return err
		}
		return g.commit()
	}

	for _, migration := range g.migrations {
		if err := g.runMigration(migration); err != nil {
			return err
		}
	}
	return g.commit()
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

	rollback := g.begin()
	defer rollback()

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
	return g.commit()
}

func (g *Migrator) rollbackMigration(m *Migration) error {
	if m.Rollback == nil {
		return errors.New("migration can not be rollback back")
	}

	if err := m.Rollback(g.tx); err != nil {
		return err
	}
	return g.tx.Exec("DELETE FROM migrations WHERE id = ?", m.ID).Error
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
	return nil
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
		logging.Errorf("Err during migration %s: %s", migration.ID, err.Error())
		return err
	}
	return g.insertMigration(migration.ID)
}

func (g *Migrator) createMigrationTableIfNotExists() error {
	// TODO: replace gorm helper
	if g.tx.Migrator().HasTable("migrations") {
		return nil
	}

	return g.tx.Exec("CREATE TABLE migrations (id VARCHAR(255) PRIMARY KEY)").Error
}

// TODO: select all values from the table once, instead of selecting each
// individually
func (g *Migrator) migrationRan(m *Migration) (bool, error) {
	var count int64
	err := g.tx.Raw(`select count(id) from migrations where id = ?`, m.ID).Scan(&count).Error
	return count > 0, err
}

func (g *Migrator) shouldInitializeSchema() (bool, error) {
	migrationRan, err := g.migrationRan(&Migration{ID: initSchemaMigrationID})
	if err != nil {
		return false, err
	}
	if migrationRan {
		return false, nil
	}

	// If the ID doesn't exist, we also want the list of migrations to be empty
	var count int64
	err = g.tx.Raw(`SELECT count(id) from migrations`).Scan(&count).Error
	return count == 0, err
}

func (g *Migrator) insertMigration(id string) error {
	return g.tx.Exec("INSERT INTO migrations (id) VALUES (?)", id).Error
}

func (g *Migrator) begin() func() {
	if g.options.UseTransaction {
		g.tx = g.db.Begin()
		return func() {
			g.tx.Rollback()
		}
	}
	g.tx = g.db
	return func() {}
}

func (g *Migrator) commit() error {
	if g.options.UseTransaction {
		return g.tx.Commit().Error
	}
	return nil
}
