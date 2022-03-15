package models

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

type Group struct {
	Model

	Name string `gorm:"uniqueIndex:idx_groups_name_provider_id,where:deleted_at is NULL"`

	ProviderID uid.ID `gorm:"uniqueIndex:idx_groups_name_provider_id,where:deleted_at is NULL"`

	Users []User `gorm:"many2many:users_groups"`
}

func (g *Group) ToAPI() *api.Group {
	return &api.Group{
		ID:         g.ID,
		Created:    g.CreatedAt.Unix(),
		Updated:    g.UpdatedAt.Unix(),
		Name:       g.Name,
		ProviderID: g.ProviderID,
	}
}

func (g *Group) PolymorphicIdentifier() uid.PolymorphicID {
	return uid.NewGroupPolymorphicID(g.ID)
}
