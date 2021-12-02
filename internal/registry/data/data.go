package data

import (
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/models"
)

func NewDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(connection, &gorm.Config{
		Logger: logger.New(
			logging.StandardErrorLog(),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.User{}, &models.Group{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Role{}, &models.RoleKubernetes{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Provider{}, &models.ProviderOkta{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Destination{}, &models.DestinationKubernetes{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Label{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Token{}, &models.APIKey{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Settings{}); err != nil {
		return nil, err
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

func add(db *gorm.DB, kind interface{}, value interface{}, condition interface{}) error {
	if err := db.Create(value).Error; err != nil {
		return err
	}

	return nil
}

func get(db *gorm.DB, kind interface{}, value interface{}, condition interface{}) error {
	if err := db.Model(kind).Preload(clause.Associations).Where(condition).First(value).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return internal.ErrNotFound
		}

		return err
	}

	return nil
}

func list(db *gorm.DB, kind interface{}, values interface{}, condition interface{}) error {
	if err := db.Model(kind).Preload(clause.Associations).Where(condition).Find(values).Error; err != nil {
		return err
	}

	return nil
}

func update(db *gorm.DB, kind interface{}, value interface{}, condition interface{}) error {
	r := db.Model(kind).Where(condition).Updates(value)
	if err := r.Error; err != nil {
		return err
	} else if r.RowsAffected == 0 {
		return internal.ErrNotFound
	}

	return nil
}

func remove(db *gorm.DB, kind interface{}, condition interface{}) error {
	return db.Model(kind).Select(clause.Associations).Where(condition).Delete(kind).Error
}

func Count(db *gorm.DB, kind interface{}, condition interface{}) (*int64, error) {
	var count int64
	if err := db.Model(kind).Where(condition).Count(&count).Error; err != nil {
		return nil, err
	}

	return &count, nil
}

func LabelSelector(db *gorm.DB, field string, labels ...string) *gorm.DB {
	if len(labels) > 0 {
		db = db.Where(
			"id IN (?)",
			db.Model(&models.Label{}).Select(field).Where("value IN (?)", labels),
		)
	}

	return db
}
