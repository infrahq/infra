package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	Model

	Name      string `gorm:"uniqueIndex:idx_organizations_name,where:deleted_at is NULL"`
	CreatedBy uid.ID
}

func (o *Organization) ToAPI() *api.Organization {
	return &api.Organization{
		ID:      o.ID,
		Created: api.Time(o.CreatedAt),
		Updated: api.Time(o.UpdatedAt),
		Name:    o.Name,
	}
}
