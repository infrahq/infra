package models

import (
	"time"

	"github.com/infrahq/infra/api"
)

type Destination struct {
	Model

	Name       string `validate:"required"`
	UniqueID   string `gorm:"uniqueIndex:idx_destinations_unique_id,where:deleted_at is NULL"`
	LastSeenAt time.Time

	ConnectionURL string
	ConnectionCA  string

	Resources CommaSeparatedStrings
	Roles     CommaSeparatedStrings
}

func (d *Destination) ToAPI() *api.Destination {
	return &api.Destination{
		ID:       d.ID,
		Created:  api.Time(d.CreatedAt),
		Updated:  api.Time(d.UpdatedAt),
		Name:     d.Name,
		UniqueID: d.UniqueID,
		Connection: api.DestinationConnection{
			URL: d.ConnectionURL,
			CA:  api.PEM(d.ConnectionCA),
		},
		Resources: d.Resources,
		Roles:     d.Roles,
		LastSeen:  api.Time(d.LastSeenAt),
	}
}
