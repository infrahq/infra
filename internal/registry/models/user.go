package models

import (
	"time"

	"github.com/infrahq/infra/internal/api"
)

type User struct {
	Model

	Email       string `gorm:"uniqueIndex:,where:deleted_at is NULL"`
	Permissions string
	LastSeenAt  time.Time // updated on when user uses a session token

	Grants    []Grant    `gorm:"many2many:users_grants"`
	Providers []Provider `gorm:"many2many:users_providers"`
	Groups    []Group    `gorm:"many2many:users_groups"`
}

func (u *User) ToAPI() api.User {
	result := api.User{
		ID:      u.ID,
		Created: u.CreatedAt.Unix(),
		Updated: u.UpdatedAt.Unix(),

		Email: u.Email,
	}

	if u.LastSeenAt.Unix() > 0 {
		result.LastSeenAt = u.LastSeenAt.Unix()
	}

	groups := make([]api.Group, 0)
	for _, g := range u.Groups {
		groups = append(groups, g.ToAPI())
	}

	if len(groups) > 0 {
		result.Groups = groups
	}

	grants := make([]api.Grant, 0)
	for _, r := range u.Grants {
		grants = append(grants, r.ToAPI())
	}

	if len(grants) > 0 {
		result.Grants = grants
	}

	providers := make([]api.Provider, 0)
	for _, r := range u.Providers {
		providers = append(providers, r.ToAPI())
	}

	if len(providers) > 0 {
		result.Providers = providers
	}

	return result
}
