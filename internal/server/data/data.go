package data

import (
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
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// NewDB creates a new database connection and runs any required database migrations
// before returning the connection. The loadDBKey function is called after
// initializing the schema, but before any migrations.
func NewDB(connection gorm.Dialector, loadDBKey func(db *gorm.DB) error) (*gorm.DB, error) {
	db, err := newRawDB(connection)
	if err != nil {
		return nil, fmt.Errorf("db conn: %w", err)
	}

	if err := preMigrate(db); err != nil {
		return nil, err
	}

	if loadDBKey != nil {
		if err := loadDBKey(db); err != nil {
			return nil, fmt.Errorf("load DB key failed: %w", err)
		}
	}

	if err = migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// newRawDB creates a new database connection without running migrations.
func newRawDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(connection, &gorm.Config{
		Logger: logging.NewDatabaseLogger(time.Second),
	})
	if err != nil {
		return nil, err
	}

	if connection.Name() == "sqlite" {
		// avoid issues with concurrent writes by telling gorm
		// not to open multiple connections in the connection pool
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("getting db driver: %w", err)
		}

		sqlDB.SetMaxOpenConns(1)
	}

	return db, nil
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

func NewPostgresDriver(connection string) (gorm.Dialector, error) {
	return postgres.Open(connection), nil
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

func get[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) (*T, error) {
	for _, selector := range selectors {
		db = selector(db)
	}
	if hasOrgID(db, new(T)) {
		db = ByOrg(getOrg(db))(db)
	}

	result := new(T)
	if err := db.Model((*T)(nil)).First(result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}

		return nil, err
	}

	return result, nil
}

func getOrg(db *gorm.DB) *models.Organization {
	org, ok := db.Statement.Context.Value("org").(*models.Organization)
	if !ok {
		panic("no org")
	}
	return org
}

func setOrg(db *gorm.DB, model interface{}) {
	if !hasOrgID(db, model) {
		return
	}
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	f := v.FieldByName("OrganizationID")
	org := getOrg(db)
	id := uid.ID(f.Int())
	if id == 0 {
		f.SetInt(int64(org.ID))
		return
	}
	if id != org.ID {
		panic("OrganizationID doesn't match org id")
	}
}

func hasOrgID(db *gorm.DB, model interface{}) bool {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	typeName := v.Type().Name()
	switch typeName {
	case "Organization", "EncryptionKey":
		return false
	}
	f := v.FieldByName("OrganizationID")
	return f.IsValid()
}

func list[T models.Modelable](db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]T, error) {
	db = db.Order(getDefaultSortFromType((*T)(nil)))
	for _, selector := range selectors {
		db = selector(db)
	}
	if hasOrgID(db, new(T)) {
		db = ByOrg(getOrg(db))(db)
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

func save[T models.Modelable](db *gorm.DB, model *T) error {
	setOrg(db, model)
	err := db.Save(model).Error
	return handleError(err)
}

func add[T models.Modelable](db *gorm.DB, model *T) error {
	setOrg(db, model)
	err := db.Create(model).Error
	return handleError(err)
}

type UniqueConstraintError struct {
	Table  string
	Column string
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

	if e.Column == "" {
		return fmt.Sprintf("a %v with that value already exists", table)
	}
	return fmt.Sprintf("a %v with that %v already exists", table, e.Column)
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

func delete[T models.Modelable](db *gorm.DB, id uid.ID) error {
	if hasOrgID(db, new(T)) {
		org := getOrg(db)
		db = db.Where("organization_id = ?", org.ID)
	}
	return db.Delete(new(T), id).Error
}

func deleteAll[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) error {
	for _, selector := range selectors {
		db = selector(db)
	}
	if hasOrgID(db, new(T)) {
		org := getOrg(db)
		db = db.Where("organization_id = ?", org.ID)
	}

	return db.Delete(new(T)).Error
}

// GlobalCount gives the count of all records, not scoped by org.
func GlobalCount[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) (int64, error) {
	for _, selector := range selectors {
		db = selector(db)
	}

	var count int64
	if err := db.Model((*T)(nil)).Count(&count).Error; err != nil {
		return -1, err
	}

	return count, nil
}

// InfraProvider is a lazy-loaded cached reference to the infra provider. The
// cache lasts for the entire lifetime of the process, so any test or test
// helper that calls InfraProvider must call InvalidateCache to clean up.
func InfraProvider(db *gorm.DB) *models.Provider {
	org := getOrg(db)
	infra, err := get[models.Provider](db, ByProviderKind(models.ProviderKindInfra), ByOrg(org))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			p := &models.Provider{Name: models.InternalInfraProviderName, Kind: models.ProviderKindInfra, Model: models.Model{OrganizationID: org.ID}}
			if err := add(db, p); err != nil {
				logging.L.Panic().Err(err).Msg("failed to create infra provider")
			}
			return p
		}
		logging.L.Panic().Err(err).Msg("failed to retrieve infra provider")
		return nil // unreachable, the line above panics
	}
	return infra
}

// InfraConnectorIdentity is a lazy-loaded reference to the connector identity.
// The cache lasts for the entire lifetime of the process, so any test or test
// helper that calls InfraConnectorIdentity must call InvalidateCache to clean up.
func InfraConnectorIdentity(db *gorm.DB) *models.Identity {
	org := getOrg(db)
	connector, err := GetIdentity(db, ByName(models.InternalInfraConnectorIdentityName), ByOrg(org))
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to retrieve connector identity")
		return nil // unreachable, the line above panics
	}

	return connector
}

// InvalidateCache is used to clear references to frequently used resources
func InvalidateCache() {
}
