package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/api"
)

type Machine struct {
	Model

	Name        string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	Description string
	Permissions string
	LastSeenAt  time.Time // updated on when machine uses a session token
}

func (m *Machine) ToAPI() *api.Machine {
	result := &api.Machine{
		ID:      m.ID,
		Created: m.CreatedAt.Unix(),
		Updated: m.UpdatedAt.Unix(),

		Name:        m.Name,
		Description: m.Description,
		Permissions: strings.Split(m.Permissions, " "),
	}

	if m.LastSeenAt.Unix() > 0 {
		result.LastSeenAt = m.LastSeenAt.Unix()
	}

	return result
}

func (m *Machine) FromAPI(from interface{}) error {
	if createRequest, ok := from.(*api.MachineCreateRequest); ok {
		m.Name = createRequest.Name
		m.Description = createRequest.Description
		m.Permissions = strings.Join(createRequest.Permissions, " ")

		return nil
	}

	return fmt.Errorf("%w: unknown request", internal.ErrBadRequest)
}
