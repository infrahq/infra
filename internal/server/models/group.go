package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Group struct {
	Model
	OrganizationMember

	Name              string
	CreatedBy         uid.ID
	CreatedByProvider uid.ID

	TotalUsers int `db:"-"`
}

func (g *Group) ToAPI() *api.Group {
	return &api.Group{
		ID:         g.ID,
		Created:    api.Time(g.CreatedAt),
		Updated:    api.Time(g.UpdatedAt),
		Name:       g.Name,
		TotalUsers: g.TotalUsers,
	}
}
