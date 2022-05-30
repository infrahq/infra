package data

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
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
		Logger: logging.ToGormLogger(logging.S),
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

func get[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) (*T, error) {
	for _, selector := range selectors {
		db = selector(db)
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

func list[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) ([]T, error) {
	for _, selector := range selectors {
		db = selector(db)
	}

	result := make([]T, 0)
	if err := db.Model((*T)(nil)).Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func save[T models.Modelable](db *gorm.DB, model *T) error {
	v := validator.New()
	if err := v.Struct(model); err != nil {
		return err
	}

	err := db.Save(model).Error
	return handleError(err)
}

func add[T models.Modelable](db *gorm.DB, model *T) error {
	v := validator.New()
	if err := v.Struct(model); err != nil {
		return err
	}

	err := db.Create(model).Error
	return handleError(err)
}

type UniqueConstraintError struct {
	Table  string
	Column string
}

func (e UniqueConstraintError) Error() string {
	return fmt.Sprintf("value for %v already exists in %v", e.Column, e.Table)
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
			// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/backend/access/nbtree/nbtinsert.c;h=f6f4af8bfe3f0876e860944e2233328f8cc3303a;hb=HEAD#l668
			// ereport(ERROR,
			//         (errcode(ERRCODE_UNIQUE_VIOLATION),
			//          errmsg("duplicate key value violates unique constraint \"%s\"",
			//                 RelationGetRelationName(rel)),
			//                 key_desc ? errdetail("Key %s already exists.",
			//                            key_desc) : 0,
			//                 errtableconstraint(heapRel,
			//                                    RelationGetRelationName(rel))));
			re := regexp.MustCompile(`Key \((?P<columnName>[[:print:]]+)\)=\([[:print:]]+\) already exists.`)
			matches := re.FindStringSubmatch(pgErr.Detail)
			if matches != nil {
				columnNameIndex := re.SubexpIndex("columnName")
				return UniqueConstraintError{Table: pgErr.TableName, Column: matches[columnNameIndex]}
			}
		}
	}

	// https://sqlite.org/src/file?name=ext/rtree/rtree.c:
	// pRtree->base.zErrMsg = sqlite3_mprintf(
	//     "UNIQUE constraint failed: %s.%s", pRtree->zName, zCol
	// );
	re := regexp.MustCompile(`UNIQUE constraint failed: (?P<tableName>[[:print:]]+)\.(?P<columnName>[[:print:]]+)`)
	matches := re.FindStringSubmatch(err.Error())
	if matches != nil {
		tableNameIndex := re.SubexpIndex("tableName")
		columnNameIndex := re.SubexpIndex("columnName")
		return UniqueConstraintError{Table: matches[tableNameIndex], Column: matches[columnNameIndex]}
	}

	return err
}

func delete[T models.Modelable](db *gorm.DB, id uid.ID) error {
	return db.Delete(new(T), id).Error
}

func deleteAll[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) error {
	for _, selector := range selectors {
		db = selector(db)
	}

	return db.Delete(new(T)).Error
}

func Count[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) (int64, error) {
	for _, selector := range selectors {
		db = selector(db)
	}

	var count int64
	if err := db.Model((*T)(nil)).Count(&count).Error; err != nil {
		return -1, err
	}

	return count, nil
}

var infraProviderCache *models.Provider

// InfraProvider is a lazy-loaded cached reference to the infra provider. The
// cache lasts for the entire lifetime of the process, so any test or test
// helper that calls InfraProvider must call InvalidateCache to clean up.
func InfraProvider(db *gorm.DB) *models.Provider {
	if infraProviderCache == nil {
		infra, err := get[models.Provider](db, ByName(models.InternalInfraProviderName))
		if err != nil {
			logging.S.Panic(err)
			return nil
		}

		infraProviderCache = infra
	}

	return infraProviderCache
}

var infraConnectorCache *models.Identity

// InfraConnectorIdentity is a lazy-loaded reference to the connector identity.
// The cache lasts for the entire lifetime of the process, so any test or test
// helper that calls InfraConnectorIdentity must call InvalidateCache to clean up.
func InfraConnectorIdentity(db *gorm.DB) *models.Identity {
	if infraConnectorCache == nil {
		connector, err := GetIdentity(db, ByName(models.InternalInfraConnectorIdentityName))
		if err != nil {
			logging.S.Panic(err)
			return nil
		}

		infraConnectorCache = connector
	}

	return infraConnectorCache
}

// InvalidateCache is used to clear references to frequently used resources
func InvalidateCache() {
	infraProviderCache = nil
	infraConnectorCache = nil
}
