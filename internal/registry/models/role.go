package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type RoleKind string

var RoleKindKubernetes RoleKind = "kubernetes"

type Role struct {
	Model
	Kind RoleKind

	DestinationID uuid.UUID
	Destination   Destination

	Groups []Group `gorm:"many2many:groups_roles"`
	Users  []User  `gorm:"many2many:users_roles"`

	Kubernetes RoleKubernetes
}

type RoleKubernetesKind string

var (
	RoleKubernetesKindRole        RoleKubernetesKind = "role"
	RoleKubernetesKindClusterRole RoleKubernetesKind = "cluster-role"
)

type RoleKubernetes struct {
	Model

	Kind      RoleKubernetesKind
	Name      string
	Namespace string

	RoleID uuid.UUID
}

func (r *Role) ToAPI() api.Role {
	result := api.Role{
		ID:      r.ID.String(),
		Created: r.CreatedAt.Unix(),
		Updated: r.UpdatedAt.Unix(),
	}

	switch r.Kind {
	case RoleKindKubernetes:
		result.Kind = api.RoleKind(r.Kubernetes.Kind)
		result.Name = r.Kubernetes.Name
		result.Namespace = r.Kubernetes.Namespace
	}

	users := make([]api.User, 0)
	for _, u := range r.Users {
		users = append(users, u.ToAPI())
	}

	if len(users) > 0 {
		result.SetUsers(users)
	}

	groups := make([]api.Group, 0)
	for _, g := range r.Groups {
		groups = append(groups, g.ToAPI())
	}

	if len(groups) > 0 {
		result.SetGroups(groups)
	}

	result.Destination = r.Destination.ToAPI()

	return result
}

func NewRole(id string) (*Role, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Role{
		Model: Model{
			ID: uuid,
		},
	}, nil
}
