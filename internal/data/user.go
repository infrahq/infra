package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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

func (u *User) BindRoles(db *gorm.DB, roleIDs ...uuid.UUID) error {
	roles, err := ListRoles(db, roleIDs)
	if err != nil {
		return err
	}

	if err := db.Model(u).Association("Roles").Replace(roles); err != nil {
		return err
	}

	return nil
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

func CreateUser(db *gorm.DB, user *User) (*User, error) {
	if err := add(db, &User{}, user, &User{}); err != nil {
		return nil, err
	}

	return user, nil
}

func CreateOrUpdateUser(db *gorm.DB, user *User, condition interface{}) (*User, error) {
	existing, err := GetUser(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateUser(db, user); err != nil {
			return nil, err
		}

		return user, nil
	}

	if err := update(db, &User{}, user, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	if err := get(db, &User{}, user, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	return user, nil
}

func GetUser(db *gorm.DB, condition interface{}) (*User, error) {
	var user User
	if err := get(db, &User{}, &user, condition); err != nil {
		return nil, err
	}

	return &user, nil
}

func ListUsers(db *gorm.DB, condition interface{}) ([]User, error) {
	users := make([]User, 0)
	if err := list(db, &User{}, &users, condition); err != nil {
		return nil, err
	}

	return users, nil
}

func DeleteUsers(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListUsers(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &User{}, ids)
	}

	return nil
}

func UserAssociations(db *gorm.DB) *gorm.DB {
	db = db.Preload("Roles.Kubernetes").Preload("Roles.Destination.Kubernetes")
	db = db.Preload("Groups.Roles.Kubernetes").Preload("Groups.Roles.Destination.Kubernetes")

	return db
}
