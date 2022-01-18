package models

import (
	"time"

	"github.com/infrahq/infra/uid"
	"gorm.io/gorm"
)

// Modelable is an interface that determines if a struct is a model. It's simply models that compose models.Model
type Modelable interface {
	IsAModel() // there's nothing specific about this function except that all Model structs will have it.
}

type Model struct {
	ID        uid.ID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (Model) IsAModel() {}

// Set an ID if one does not already exist. Unfortunately, we can use `gorm:"default"`
// tags since the ID must be dynamically generated and not all databases support UUID generation
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = uid.New()
	}

	return nil
}
