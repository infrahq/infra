package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type User struct {
	Model

	Name  string
	Email string

	Roles     []Role     `gorm:"many2many:users_roles"`
	Providers []Provider `gorm:"many2many:users_providers"`
	Groups    []Group    `gorm:"many2many:users_groups"`
}

func (u *User) ToAPI() api.User {
	result := api.User{
		Id:      u.ID.String(),
		Created: u.CreatedAt.Unix(),
		Updated: u.UpdatedAt.Unix(),

		Email: u.Email,
	}

	for _, g := range u.Groups {
		result.Groups = append(result.Groups, g.ToAPI())
	}

	for _, r := range u.Roles {
		result.Roles = append(result.Roles, r.ToAPI())
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
