package models

import (
	"fmt"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
)

type Machine struct {
	Model

	Name        string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	Description string
	LastSeenAt  time.Time // updated on when machine uses a session token
}

func (m *Machine) ToAPI() *api.Machine {
	result := &api.Machine{
		ID:      m.ID,
		Created: m.CreatedAt.Unix(),
		Updated: m.UpdatedAt.Unix(),

		Name:        m.Name,
		Description: m.Description,
	}

	if m.LastSeenAt.Unix() > 0 {
		result.LastSeenAt = m.LastSeenAt.Unix()
	}

	return result
}

func (m *Machine) FromAPI(from interface{}) error {
	if createRequest, ok := from.(*api.CreateMachineRequest); ok {
		m.Name = createRequest.Name
		m.Description = createRequest.Description

		return nil
	}

	return fmt.Errorf("%w: unknown request", internal.ErrBadRequest)
}

func (m *Machine) PolymorphicIdentifier() uid.PolymorphicID {
	return uid.NewMachinePolymorphicID(m.ID)
}
