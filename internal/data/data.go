package data

import (
	"errors"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/infrahq/infra/internal"
)

type Model struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Set an ID if one does not already exist. Unfortunately, we can use `gorm:"default"`
// tags since the ID must be dynamically generated and not all databases support UUID generation
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = NewID()
	}

	return nil
}

// Generate new UUIDv1
func NewID() uuid.UUID {
	return uuid.Must(uuid.NewUUID())
}

func NewDB(connection gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(connection, &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Group{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Role{}, &RoleKubernetes{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Provider{}, &ProviderOkta{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Destination{}, &DestinationKubernetes{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Label{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Token{}, &APIKey{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Settings{}); err != nil {
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
			db.Model(&Label{}).Select(field).Where("value IN (?)", labels),
		)
	}

	return db
}
