package models

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
)

type Grant struct {
	Model

	Role     string `gorm:"uniqueIndex:idx_grants_role_resource,where:deleted_at is NULL"`
	Resource `gorm:"embeddedPrefix:resource_"`

	Groups []Group `gorm:"many2many:groups_grants"`
	Users  []User  `gorm:"many2many:users_grants"`

	Labels []Label `gorm:"many2many:grant_labels"`
}

type Resource struct {
	Kind string `gorm:"uniqueIndex:idx_grants_role_resource"`
	Name string `gorm:"uniqueIndex:idx_grants_role_resource"`
	Path string `gorm:"uniqueIndex:idx_grants_role_resource"`
}

func (g *Grant) ToAPI() api.Grant {
	result := api.Grant{
		ID:      g.ID.String(),
		Created: g.CreatedAt.Unix(),
		Updated: g.UpdatedAt.Unix(),
		Resource: api.GrantResource{
			Kind: g.Resource.Kind,
			Name: g.Resource.Name,
			Path: g.Resource.Path,
		},
	}

	labels := make([]string, 0)
	for _, l := range g.Labels {
		labels = append(labels, l.Value)
	}

	result.SetLabels(labels)

	if len(g.Role) > 0 {
		result.SetRole(g.Role)
	}

	users := make([]api.User, 0)
	for _, u := range g.Users {
		users = append(users, u.ToAPI())
	}

	if len(users) > 0 {
		result.SetUsers(users)
	}

	groups := make([]api.Group, 0)
	for _, g := range g.Groups {
		groups = append(groups, g.ToAPI())
	}

	if len(groups) > 0 {
		result.SetGroups(groups)
	}

	return result
}

func (g *Grant) FromAPI(from interface{}) error {
	if request, ok := from.(*api.GrantRequest); ok {
		g.Role = request.GetRole()
		g.Resource = Resource{
			Kind: request.Resource.Kind,
			Name: request.Resource.Name,
			Path: request.Resource.Path,
		}

		for _, l := range request.GetLabels() {
			g.Labels = append(g.Labels, Label{Value: l})
		}

		for _, id := range request.GetUsers() {
			user, err := NewUser(id)
			if err != nil {
				logging.S.Infof("invalid user ID: %s", id)
				continue
			}

			g.Users = append(g.Users, *user)
		}

		for _, id := range request.GetGroups() {
			group, err := NewGroup(id)
			if err != nil {
				logging.S.Infof("invalid group ID: %s", id)
				continue
			}

			g.Groups = append(g.Groups, *group)
		}

		return nil
	}

	return fmt.Errorf("unknown grant model")
}

func NewGrant(id string) (*Grant, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Grant{
		Model: Model{
			ID: uuid,
		},
	}, nil
}
