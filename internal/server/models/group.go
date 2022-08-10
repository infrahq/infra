package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Group struct {
	Model

	Name              string `gorm:"uniqueIndex:idx_groups_name,where:deleted_at is NULL"`
	CreatedBy         uid.ID
	CreatedByProvider uid.ID

	Identities []Identity `gorm:"many2many:identities_groups" db:"-"`

	TotalUsers int `gorm:"-:all" db:"-"`
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

func (g *Group) PolyID() uid.PolymorphicID {
	return uid.NewGroupPolymorphicID(g.ID)
}
