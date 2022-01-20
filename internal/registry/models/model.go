package models

import (
	"time"

	"github.com/infrahq/infra/uuid"
	"gorm.io/gorm"
)

type Model struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

// Set an ID if one does not already exist. Unfortunately, we can use `gorm:"default"`
// tags since the ID must be dynamically generated and not all databases support UUID generation
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = NewID()
	}

	return nil
}

// Generate new UUIDv1
func NewID() uuid.UUID {
	return uuid.New()
}
