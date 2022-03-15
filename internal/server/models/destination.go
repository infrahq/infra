package models

import (
	"github.com/infrahq/infra/api"
)

type Destination struct {
	Model

	Name     string `validate:"required"`
	UniqueID string `gorm:"uniqueIndex:,where:deleted_at is NULL"`

	ConnectionURL string
	ConnectionCA  string
}

func (d *Destination) ToAPI() *api.Destination {
	return &api.Destination{
		ID:       d.ID,
		Created:  d.CreatedAt.Unix(),
		Updated:  d.UpdatedAt.Unix(),
		Name:     d.Name,
		UniqueID: d.UniqueID,
		Connection: api.DestinationConnection{
			URL: d.ConnectionURL,
			CA:  d.ConnectionCA,
		},
	}
}
