package models

import (
	"database/sql"
	"time"

	"github.com/infrahq/infra/uid"
)

const CreatedBySystem = 1

type Model struct {
	ID uid.ID
	// CreatedAt is set to time.Now on insert and should not be changed after
	// insert.
	CreatedAt time.Time
	// UpdatedAt is set to time.Now on insert and update.
	UpdatedAt time.Time
	// DeletedAt is set to time.Now when the row is soft-deleted.
	DeletedAt sql.NullTime
}

func (m Model) Primary() uid.ID {
	return m.ID
}

func (m *Model) OnInsert() error {
	if m.ID == 0 {
		m.ID = uid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	m.UpdatedAt = m.CreatedAt
	return nil
}

func (m *Model) OnUpdate() error {
	m.UpdatedAt = time.Now()
	return nil
}
