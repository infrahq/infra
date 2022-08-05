package models

import (
	"time"

	"github.com/infrahq/infra/api"
)

type Destination struct {
	Model

	Name       string
	UniqueID   string
	LastSeenAt time.Time

	Version string

	ConnectionURL string
	ConnectionCA  string

	Resources CommaSeparatedStrings
	Roles     CommaSeparatedStrings
}

func (d *Destination) ToAPI() *api.Destination {
	connected := false
	// TODO: this should be configurable
	// https://github.com/infrahq/infra/issues/2505
	if time.Since(d.LastSeenAt) < 5*time.Minute {
		connected = true
	}

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
		Connected: connected,
		Version:   d.Version,
	}
}
