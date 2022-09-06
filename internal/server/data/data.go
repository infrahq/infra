package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data/migrator"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// NewDB creates a new database connection and runs any required database migrations
// before returning the connection. The loadDBKey function is called after
// initializing the schema, but before any migrations.
func NewDB(connection gorm.Dialector, loadDBKey func(db GormTxn) error) (*DB, error) {
	db, err := newRawDB(connection)
	if err != nil {
		return nil, fmt.Errorf("db conn: %w", err)
	}
	dataDB := &DB{DB: db}

	opts := migrator.Options{
		InitSchema: initializeSchema,
		LoadKey: func(tx migrator.DB) error {
			if loadDBKey == nil {
				return nil
			}
			// TODO: use the passed in tx instead of dataDB once the queries
			// used by loadDBKey are ported to sql
			return loadDBKey(dataDB)
		},
	}
	m := migrator.New(dataDB, opts, migrations())
	if err := m.Migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
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

// TODO: accept sql.TxOptions when we remove gorm
func (d *DB) Begin(ctx context.Context) (*Transaction, error) {
	tx := d.DB.WithContext(ctx).Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}
	return &Transaction{DB: tx}, nil
}

type WriteTxn interface {
	ReadTxn
	Exec(sql string, values ...interface{}) (sql.Result, error)
}

type ReadTxn interface {
	DriverName() string
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	OrganizationID() uid.ID
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
	orgID uid.ID
}

func (t Transaction) DriverName() string {
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

func (t *Transaction) Rollback() error {
	return t.DB.Rollback().Error
}

func (t *Transaction) Commit() error {
	return t.DB.Commit().Error
}

// WithOrgID returns a copy of the Transaction with the OrganizationID set to
// orgID.
func (t *Transaction) WithOrgID(orgID uid.ID) *Transaction {
	newTxn := *t
	newTxn.orgID = orgID
	return &newTxn
}

// newRawDB creates a new database connection without running migrations.
func newRawDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(connection, &gorm.Config{
		Logger: logging.NewDatabaseLogger(time.Second),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting db driver: %w", err)
	}

	if connection.Name() == "sqlite" {
		// avoid issues with concurrent writes by telling gorm
		// not to open multiple connections in the connection pool
		sqlDB.SetMaxOpenConns(1)
	} else {
		// TODO: make these configurable from server config
		sqlDB.SetMaxIdleConns(900)
		sqlDB.SetMaxOpenConns(1000)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	return db, nil
}

const defaultOrganizationID = 1000

func initialize(db *DB) error {
	org, err := GetOrganization(db, ByID(defaultOrganizationID))
	switch {
	case errors.Is(err, internal.ErrNotFound):
		org = &models.Organization{
			Model:     models.Model{ID: defaultOrganizationID},
			Name:      "Default",
			CreatedBy: models.CreatedBySystem,
		}
		if err := CreateOrganization(db, org); err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
	case err != nil:
		return fmt.Errorf("failed to get default organization: %w", err)
	}

	db.DefaultOrg = org
	db.DefaultOrgSettings, err = getSettingsForOrg(db, org.ID)
	if err != nil {
		return fmt.Errorf("getting settings: %w", err)
	}
	return nil
}

func NewSQLiteDriver(connection string) (gorm.Dialector, error) {
	if !strings.HasPrefix(connection, "file::memory") {
		if err := os.MkdirAll(path.Dir(connection), os.ModePerm); err != nil {
			return nil, err
		}
	}
	uri, err := url.Parse(connection)
	if err != nil {
		return nil, err
	}
	query := uri.Query()
	query.Add("_journal_mode", "WAL")
	uri.RawQuery = query.Encode()
	connection = uri.String()

	return sqlite.Open(connection), nil
}

func getDefaultSortFromType(t interface{}) string {
	ty := reflect.TypeOf(t).Elem()
	if _, ok := ty.FieldByName("Name"); ok {
		return "name ASC"
	}

	if _, ok := ty.FieldByName("Email"); ok {
		// foreign key relations, such as provider users in this case, may not have the default ID
		return "email ASC"
	}

	return "id ASC"
}

func get[T models.Modelable](tx GormTxn, selectors ...SelectorFunc) (*T, error) {
	db := tx.GormDB()
	for _, selector := range selectors {
		db = selector(db)
	}

	result := new(T)
	if isOrgMember(result) {
		db = ByOrgID(tx.OrganizationID())(db)
	}

	if err := db.Model((*T)(nil)).First(result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}

		return nil, err
	}

	return result, nil
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

func list[T models.Modelable](tx GormTxn, p *Pagination, selectors ...SelectorFunc) ([]T, error) {
	db := tx.GormDB()
	db = db.Order(getDefaultSortFromType((*T)(nil)))
	for _, selector := range selectors {
		db = selector(db)
	}
	if isOrgMember(new(T)) {
		db = ByOrgID(tx.OrganizationID())(db)
	}

	if p != nil {
		var count int64
		if err := db.Model((*T)(nil)).Count(&count).Error; err != nil {
			return nil, err
		}
		p.SetTotalCount(int(count))

		db = ByPagination(*p)(db)
	}

	result := make([]T, 0)
	if err := db.Model((*T)(nil)).Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func save[T models.Modelable](tx GormTxn, model *T) error {
	db := tx.GormDB()
	setOrg(tx, model)
	err := db.Save(model).Error
	return handleError(err)
}

func add[T models.Modelable](tx GormTxn, model *T) error {
	db := tx.GormDB()
	setOrg(tx, model)

	var err error
	if tx.DriverName() == "postgres" {
		// failures on postgres need to be rolled back in order to
		// continue using the same transaction
		db.SavePoint("beforeCreate")
		err = db.Create(model).Error
		if err != nil {
			db.RollbackTo("beforeCreate")
		}
	} else {
		err = db.Create(model).Error
	}
	return handleError(err)
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

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			// constraintFields maps the name of a unique constraint, to the
			// user facing name of that field.
			constraintFields := map[string]string{
				"idx_identities_name":         "name",
				"idx_groups_name":             "name",
				"idx_providers_name":          "name",
				"idx_access_keys_name":        "name",
				"idx_destinations_unique_id":  "uniqueId",
				"idx_access_keys_key_id":      "keyId",
				"idx_credentials_identity_id": "identityId",
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

func delete[T models.Modelable](tx GormTxn, id uid.ID) error {
	db := tx.GormDB()
	if isOrgMember(new(T)) {
		db = ByOrgID(tx.OrganizationID())(db)
	}
	return db.Delete(new(T), id).Error
}

func deleteAll[T models.Modelable](tx GormTxn, selectors ...SelectorFunc) error {
	db := tx.GormDB()
	for _, selector := range selectors {
		db = selector(db)
	}
	if isOrgMember(new(T)) {
		db = ByOrgID(tx.OrganizationID())(db)
	}

	return db.Delete(new(T)).Error
}

// GlobalCount gives the count of all records, not scoped by org.
func GlobalCount[T models.Modelable](tx GormTxn, selectors ...SelectorFunc) (int64, error) {
	db := tx.GormDB()
	for _, selector := range selectors {
		db = selector(db)
	}

	var count int64
	if err := db.Model((*T)(nil)).Count(&count).Error; err != nil {
		return -1, err
	}

	return count, nil
}

// InfraProvider returns the infra provider for the organization set in the db
// context.
func InfraProvider(db GormTxn) *models.Provider {
	infra, err := get[models.Provider](db, ByProviderKind(models.ProviderKindInfra), ByOrgID(db.OrganizationID()))
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to retrieve infra provider")
		return nil // unreachable, the line above panics
	}
	return infra
}

// InfraConnectorIdentity returns the connector identity for the organization set
// in the db context.
func InfraConnectorIdentity(db GormTxn) *models.Identity {
	connector, err := GetIdentity(db, ByName(models.InternalInfraConnectorIdentityName), ByOrgID(db.OrganizationID()))
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to retrieve connector identity")
		return nil // unreachable, the line above panics
	}
	return connector
}
