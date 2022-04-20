package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/uid"
)

// Modelable is an interface that determines if a struct is a model. It's simply models that compose models.Model
type Modelable interface {
	IsAModel() // there's nothing specific about this function except that all Model structs will have it.
}

type Model struct {
	ID uid.ID
	// CreatedAt is set by GORM to time.Now when a record is first created.
	// See https://gorm.io/docs/conventions.html#Timestamp-Tracking
	CreatedAt time.Time
	// CreatedAt is set by GORM to time.Now when a record is updated.
	// See https://gorm.io/docs/conventions.html#Timestamp-Tracking
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (Model) IsAModel() {}

// BeforeCreate sets an ID if one does not already exist. Unfortunately, we can use `gorm:"default"`
// tags since the ID must be dynamically generated and not all databases support UUID generation.
func (m *Model) BeforeCreate(_ *gorm.DB) error {
	if m.ID == 0 {
		m.ID = uid.New()
	}

	return nil
}
