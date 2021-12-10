package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type Group struct {
	Model

	Name string `gorm:"uniqueIndex:,where:deleted_at is NULL"`

	Grants    []Grant    `gorm:"many2many:groups_grants"`
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

	users := make([]api.User, 0)
	for _, u := range g.Users {
		users = append(users, u.ToAPI())
	}

	if len(users) > 0 {
		result.SetUsers(users)
	}

	grants := make([]api.Grant, 0)
	for _, r := range g.Grants {
		grants = append(grants, r.ToAPI())
	}

	if len(grants) > 0 {
		result.SetGrants(grants)
	}

	providers := make([]api.Provider, 0)
	for _, r := range g.Providers {
		providers = append(providers, r.ToAPI())
	}

	if len(providers) > 0 {
		result.SetProviders(providers)
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
