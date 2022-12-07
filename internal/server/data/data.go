package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/migrator"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

type NewDBOptions struct {
	DSN string

	EncryptionKeyProvider EncryptionKeyProvider
	RootKeyID             string

	MaxOpenConnections int
	MaxIdleConnections int
	MaxIdleTimeout     time.Duration
}

// NewDB creates a new database connection and runs any required database migrations
// before returning the connection. The loadDBKey function is called after
// initializing the schema, but before any migrations.
func NewDB(dbOpts NewDBOptions) (*DB, error) {
	db, err := newRawDB(dbOpts)
	if err != nil {
		return nil, fmt.Errorf("db conn: %w", err)
	}
	dataDB := &DB{DB: db}
	tx, err := dataDB.Begin(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	opts := migrator.Options{
		InitSchema: initializeSchema,
		LoadKey: func(tx migrator.DB) error {
			if dbOpts.EncryptionKeyProvider == nil {
				return nil
			}
			return loadDBKey(tx, dbOpts.EncryptionKeyProvider, dbOpts.RootKeyID)
		},
	}
	m := migrator.New(tx, opts, migrations())
	if err := m.Migrate(); err != nil {
		if err := tx.Rollback(); err != nil {
			logging.L.Warn().Err(err).Msg("failed to rollback")
		}
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit migrations: %w", err)
	}

	if err := initialize(dataDB); err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	return dataDB, nil
}

// DB wraps the underlying database and provides access to the default org,
// and settings.
type DB struct {
	*gorm.DB // embedded for now to minimize the diff

	DefaultOrg *models.Organization
	// DefaultOrgSettings are the settings for DefaultOrg
	DefaultOrgSettings *models.Settings
}

func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database conn to close: %w", err)
	}
	return sqlDB.Close()
}

func (d *DB) SQLdb() *sql.DB {
	sqlDB, err := d.DB.DB()
	if err != nil {
		panic("DB must have an sql.DB ConnPool")
	}
	return sqlDB
}

func (d *DB) DriverName() string {
	return d.Dialector.Name()
}

func (d *DB) Exec(query string, args ...any) (sql.Result, error) {
	db := d.DB.Exec(query, args...)
	return driver.RowsAffected(db.RowsAffected), db.Error
}

func (d *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return d.DB.Raw(query, args...).Rows()
}

func (d *DB) QueryRow(query string, args ...any) *sql.Row {
	return d.DB.Raw(query, args...).Row()
}

func (d *DB) OrganizationID() uid.ID {
	// FIXME: this is a hack to keep our tests passing. The db should not
	// be scoped to an org ID.
	return d.DefaultOrg.ID
}

func (d *DB) GormDB() *gorm.DB {
	return d.DB
}

func (d *DB) Begin(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	tx := d.DB.WithContext(ctx).Begin(opts)
	if err := tx.Error; err != nil {
		return nil, err
	}
	return &Transaction{DB: tx, completed: new(atomic.Bool)}, nil
}

// GormTxn is used as a shim in preparation for removing gorm.
type GormTxn interface {
	WriteTxn

	// GormDB returns the underlying reference to the gorm.DB struct.
	// Do not use this in new code! Instead, write SQL using the stdlib\
	// interface of Query, QueryRow, and Exec.
	GormDB() *gorm.DB
}

type Transaction struct {
	*gorm.DB
	orgID     uid.ID
	completed *atomic.Bool
}

func (t *Transaction) DriverName() string {
	return t.Dialector.Name()
}

func (t *Transaction) OrganizationID() uid.ID {
	return t.orgID
}

func (t *Transaction) Exec(query string, args ...any) (sql.Result, error) {
	db := t.DB.Exec(query, args...)
	return driver.RowsAffected(db.RowsAffected), db.Error
}

func (t *Transaction) Query(query string, args ...any) (*sql.Rows, error) {
	return t.DB.Raw(query, args...).Rows()
}

func (t *Transaction) QueryRow(query string, args ...any) *sql.Row {
	return t.DB.Raw(query, args...).Row()
}

func (t *Transaction) GormDB() *gorm.DB {
	return t.DB
}

// Rollback the transaction. If the transaction was already committed then do
// nothing.
func (t *Transaction) Rollback() error {
	if t.completed.Load() {
		return nil
	}
	err := t.DB.Rollback().Error
	if err == nil {
		t.completed.Store(true)
	}
	return err
}

func (t *Transaction) Begin(opts ...*sql.TxOptions) error {
	t.DB = t.DB.Session(&gorm.Session{NewDB: true})
	db := t.DB.Begin(opts...)
	if db.Error != nil {
		return db.Error
	}
	t.completed.Store(false)
	t.DB = db
	return nil
}

func (t *Transaction) Commit() error {
	err := t.DB.Commit().Error
	if err == nil {
		t.completed.Store(true)
	}
	return err
}

// WithOrgID returns a shallow copy of the Transaction with the OrganizationID
// set to orgID. Note that the underlying database transaction and commit state
// is shared with the new copy.
func (t *Transaction) WithOrgID(orgID uid.ID) *Transaction {
	newTxn := *t
	newTxn.orgID = orgID
	return &newTxn
}

// newRawDB creates a new database connection without running migrations.
func newRawDB(options NewDBOptions) (*gorm.DB, error) {
	if options.DSN == "" {
		return nil, fmt.Errorf("missing postgres dsn")
	}

	db, err := gorm.Open(postgres.Open(options.DSN), &gorm.Config{
		Logger: logging.NewDatabaseLogger(time.Second),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting db driver: %w", err)
	}

	sqlDB.SetMaxOpenConns(options.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(options.MaxIdleConnections)
	sqlDB.SetConnMaxIdleTime(options.MaxIdleTimeout)

	return db, nil
}

const defaultOrganizationID = 1000

func initialize(db *DB) error {
	tx, err := db.Begin(context.TODO(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	org, err := GetOrganization(tx, GetOrganizationOptions{ByID: defaultOrganizationID})
	switch {
	case errors.Is(err, internal.ErrNotFound):
		org = &models.Organization{
			Model:     models.Model{ID: defaultOrganizationID},
			Name:      "Default",
			CreatedBy: models.CreatedBySystem,
		}
		if err := CreateOrganization(tx, org); err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
	case err != nil:
		return fmt.Errorf("failed to get default organization: %w", err)
	}

	db.DefaultOrg = org
	db.DefaultOrgSettings, err = getSettingsForOrg(tx, org.ID)
	if err != nil {
		return fmt.Errorf("getting settings: %w", err)
	}
	return tx.Commit()
}

// setOrg checks if model is an organization member, and sets the organizationID
// from the transaction when it is an organization member.
func setOrg(tx ReadTxn, model any) {
	member, ok := model.(orgMember)
	if !ok {
		return
	}

	member.SetOrganizationID(tx)
}

type orgMember interface {
	IsOrganizationMember()
	SetOrganizationID(source models.OrganizationIDSource)
}

func isOrgMember(model any) bool {
	_, ok := model.(orgMember)
	return ok
}

type UniqueConstraintError struct {
	Table  string
	Column string
}

// these are tables whose names need the 'an' article rather than 'a'
var anArticleTableName = map[string]bool{
	"access key":   true,
	"organization": true,
}

func (e UniqueConstraintError) Error() string {
	table := e.Table
	switch table {
	case "":
		return "value already exists"
	case "identities":
		table = "user"
	case "access_keys":
		table = "access key"
	default:
		table = strings.TrimSuffix(table, "s")
	}

	article := "a"
	if anArticleTableName[table] {
		article = "an"
	}

	if e.Column == "" {
		return fmt.Sprintf("%s %v with that value already exists", article, table)
	}
	return fmt.Sprintf("%s %v with that %v already exists", article, table, e.Column)
}

// handleError looks for well known DB errors. If the error is recognized it
// is translated into a UniqueConstraintError so that calling code can
// inspect the error.
func handleError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return internal.ErrNotFound
	case errors.Is(err, sql.ErrNoRows):
		return internal.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			// constraintFields maps the name of a unique constraint, to the
			// user facing name of that field.
			constraintFields := map[string]string{
				"idx_identities_name":         "name",
				"idx_identities_verified":     "verificationToken",
				"idx_groups_name":             "name",
				"idx_providers_name":          "name",
				"idx_access_keys_name":        "name",
				"idx_destinations_unique_id":  "uniqueID",
				"idx_access_keys_key_id":      "keyId",
				"idx_credentials_identity_id": "identityID",
				"idx_organizations_domain":    "domain",
			}

			columnName := constraintFields[pgErr.ConstraintName]
			return UniqueConstraintError{Table: pgErr.TableName, Column: columnName}
		}
	}

	// https://sqlite.org/src/file?name=ext/rtree/rtree.c:
	// pRtree->base.zErrMsg = sqlite3_mprintf(
	//     "UNIQUE constraint failed: %s.%s", pRtree->zName, zCol
	// );
	if strings.HasPrefix(err.Error(), "UNIQUE constraint failed:") {
		fields := strings.FieldsFunc(err.Error(), func(r rune) bool {
			return unicode.IsSpace(r) || r == '.'
		})

		// fields = [UNIQUE, constraint, failed:, <table>, column>]
		switch len(fields) {
		case 5, 7, 9, 11:
			col := fields[4]
			i := 6
			for i < len(fields) {
				col += fields[i]
				i += 2
			}
			return UniqueConstraintError{
				Table:  fields[3],
				Column: col,
			}
		default:
			logging.Warnf("unhandled unique constraint error format: %q", err.Error())

			return UniqueConstraintError{}
		}
	}

	return err
}

// InfraProvider returns the infra provider for the organization set in the tx.
func InfraProvider(tx ReadTxn) *models.Provider {
	infra, err := GetProvider(tx, GetProviderOptions{KindInfra: true})
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to retrieve infra provider")
	}
	return infra
}

// InfraConnectorIdentity returns the connector identity for the organization set
// in the db context.
func InfraConnectorIdentity(db GormTxn) *models.Identity {
	connector, err := GetIdentity(db, GetIdentityOptions{ByName: models.InternalInfraConnectorIdentityName})
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to retrieve connector identity")
	}
	return connector
}
