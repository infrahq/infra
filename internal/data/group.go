package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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
		Id:      g.ID.String(),
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

	// for _, p := range g.Providers {
	// 	result.Providers = append(result.Providers, p.ToAPI())
	// }

	return result
}

func (g *Group) BindUsers(db *gorm.DB, users ...User) error {
	if err := db.Model(g).Association("Users").Replace(users); err != nil {
		return err
	}

	return nil
}

func (g *Group) BindRoles(db *gorm.DB, roleIDs ...uuid.UUID) error {
	roles, err := ListRoles(db, roleIDs)
	if err != nil {
		return err
	}

	if err := db.Model(g).Association("Roles").Replace(roles); err != nil {
		return err
	}

	return nil
}

func CreateGroup(db *gorm.DB, group *Group) (*Group, error) {
	if err := add(db, &Group{}, group, &Group{}); err != nil {
		return nil, err
	}

	return group, nil
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

func CreateOrUpdateGroup(db *gorm.DB, group *Group, condition interface{}) (*Group, error) {
	existing, err := GetGroup(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateGroup(db, group); err != nil {
			return nil, err
		}

		return group, nil
	}

	if err := update(db, &Group{}, group, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	if err := get(db, &Group{}, group, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	return group, nil
}

func GetGroup(db *gorm.DB, condition interface{}) (*Group, error) {
	var group Group
	if err := get(db, &Group{}, &group, condition); err != nil {
		return nil, err
	}

	return &group, nil
}

func ListGroups(db *gorm.DB, condition interface{}) ([]Group, error) {
	groups := make([]Group, 0)
	if err := list(db, &Group{}, &groups, condition); err != nil {
		return nil, err
	}

	return groups, nil
}

func DeleteGroups(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListGroups(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &Group{}, ids)
	}

	return nil
}

func GroupAssociations(db *gorm.DB) *gorm.DB {
	db = db.Preload("Roles.Kubernetes").Preload("Roles.Destination.Kubernetes")
	db = db.Preload("Users.Roles.Kubernetes").Preload("Users.Roles.Destination.Kubernetes")

	return db
}
