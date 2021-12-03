package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type User struct {
	Model

	Name        string
	Email       string
	Permissions string

	Grants     []Grant     `gorm:"many2many:users_grants"`
	Providers []Provider `gorm:"many2many:users_providers"`
	Groups    []Group    `gorm:"many2many:users_groups"`
}

func (u *User) ToAPI() api.User {
	result := api.User{
		ID:      u.ID.String(),
		Created: u.CreatedAt.Unix(),
		Updated: u.UpdatedAt.Unix(),

		Email: u.Email,
	}

	groups := make([]api.Group, 0)
	for _, g := range u.Groups {
		groups = append(groups, g.ToAPI())
	}

	if len(groups) > 0 {
		result.SetGroups(groups)
	}

	grants := make([]api.Grant, 0)
	for _, r := range u.Grants {
		grants = append(grants, r.ToAPI())
	}

	if len(grants) > 0 {
		result.SetGrants(grants)
	}

	providers := make([]api.Provider, 0)
	for _, r := range u.Providers {
		providers = append(providers, r.ToAPI())
	}

	if len(providers) > 0 {
		result.SetProviders(providers)
	}

	return result
}

func NewUser(id string) (*User, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &User{
		Model: Model{
			ID: uuid,
		},
	}, nil
}
