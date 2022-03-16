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
	"gorm.io/gorm/logger"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func NewDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(connection, &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, err
	}

	if connection.Name() == "sqlite" {
		// avoid issues with concurrent writes by telling gorm
		// not to open multiple connections in the connection pool
		db2, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("getting db driver: %w", err)
		}
		db2.SetMaxOpenConns(1)
	}

	tables := []interface{}{
		&models.User{},
		&models.Machine{},
		&models.Group{},
		&models.Grant{},
		&models.Provider{},
		&models.ProviderToken{},
		&models.Destination{},
		&models.AccessKey{},
		&models.Settings{},
		&models.EncryptionKey{},
		&models.TrustedCertificate{},
		&models.RootCertificate{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return nil, err
		}
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
	db2 := db
	for _, selector := range selectors {
		db2 = selector(db2)
	}

	model := new(T)
	if err := db2.Model((*T)(nil)).First(model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}

		return nil, err
	}

	return model, nil
}

func list[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) ([]T, error) {
	db2 := db
	for _, selector := range selectors {
		db2 = selector(db2)
	}

	models := make([]T, 0)
	if err := db2.Model((*T)(nil)).Find(&models).Error; err != nil {
		return nil, err
	}

	return models, nil
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

func load[T models.Modelable](db *gorm.DB, id uid.ID) (*T, error) {
	model := new(T)
	if err := db.Model((*T)(nil)).Where("id = ?", id).First(model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}

		return nil, err
	}

	return model, nil
}

func add[T models.Modelable](db *gorm.DB, model *T) error {
	v := validator.New()
	if err := v.Struct(model); err != nil {
		return err
	}

	if err := db.Create(model).Error; err != nil {
		// HACK: Compare error string instead of checking sqlite3.Error which requires
		//       possibly cross-compiling go-sqlite3. Not worth.
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

func delete[T models.Modelable](db *gorm.DB, id uid.ID) error {
	return db.Delete(new(T), id).Error
}

func deleteAll[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) error {
	db2 := db
	for _, selector := range selectors {
		db2 = selector(db2)
	}

	return db2.Delete(new(T)).Error
}

func Count[T models.Modelable](db *gorm.DB, selectors ...SelectorFunc) (*int64, error) {
	db2 := db
	for _, selector := range selectors {
		db2 = selector(db2)
	}

	var count int64
	if err := db.Model((*T)(nil)).Count(&count).Error; err != nil {
		return nil, err
	}

	return &count, nil
}

// bindAssociations replaces the association (U) of the entity (T)
func bindAssociations[T models.Modelable, U models.Modelable](db *gorm.DB, model *T, association string, replacements []U) error {
	if err := db.Model(model).Association(association).Replace(replacements); err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}

// appendAssociation adds an association (U) to the associations for the entity (T)
func appendAssociation[T models.Modelable, U models.Modelable](db *gorm.DB, model *T, association string, associatedEntity *U) error {
	if err := db.Model(model).Association(association).Append(associatedEntity); err != nil {
		return fmt.Errorf("append: %w", err)
	}
	return nil
}
