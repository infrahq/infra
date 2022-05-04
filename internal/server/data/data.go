package data

import (
	"errors"
	"fmt"
	"os"
	"path"
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
// before returning the connection.
func NewDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := NewRawDB(connection)
	if err != nil {
		return nil, fmt.Errorf("db conn: %w", err)
	}

	if err = Migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// NewRawDB creates a new database connection without running migrations
func NewRawDB(connection gorm.Dialector) (*gorm.DB, error) {
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

	if err := db.Save(model).Error; err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed:") {
			return fmt.Errorf("%w: %s", internal.ErrDuplicate, err)
		}

		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == pgerrcode.UniqueViolation {
				return fmt.Errorf("%w: %s", internal.ErrDuplicate, err)
			}
		}

		return err
	}

	return nil
}

func add[T models.Modelable](db *gorm.DB, model *T) error {
	v := validator.New()
	if err := v.Struct(model); err != nil {
		return err
	}

	if err := db.Create(model).Error; err != nil {
		if isUniqueConstraintViolation(err) {
			return fmt.Errorf("%w: %s", internal.ErrDuplicate, err)
		}

		return err
	}

	return nil
}

func isUniqueConstraintViolation(err error) bool {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		return pgerr.Code == pgerrcode.UniqueViolation
	}

	if strings.HasPrefix(err.Error(), "UNIQUE constraint failed:") {
		return true
	}

	return false
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

// InfraProvider is a lazy-loaded cached reference to the infra provider, since it's used in a lot of places
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

// InfraConnectorIdentity is a lazy-loaded reference to the connector identity
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
