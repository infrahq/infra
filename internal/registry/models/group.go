package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type Group struct {
	Model

	Name string

	Roles     []Role     `gorm:"many2many:groups_roles"`
	Providers []Provider `gorm:"many2many:groups_providers"`
	Users     []User     `gorm:"many2many:users_groups"`
}

func (g *Group) ToAPI() api.Group {
	result := api.Group{
		ID:      g.ID.String(),
		Created: g.CreatedAt.Unix(),
		Updated: g.UpdatedAt.Unix(),

		Name: g.Name,
	}

	for _, u := range g.Users {
		result.Users = append(result.Users, u.ToAPI())
	}

	for _, r := range g.Roles {
		result.Roles = append(result.Roles, r.ToAPI())
	}

	return result
}

func NewGroup(id string) (*Group, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Group{
		Model: Model{
			ID: uuid,
		},
	}, nil
}
