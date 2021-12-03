package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type GrantKind string

var GrantKindKubernetes GrantKind = "kubernetes"

type Grant struct {
	Model
	Kind GrantKind

	DestinationID uuid.UUID
	Destination   Destination

	Groups []Group `gorm:"many2many:groups_grants"`
	Users  []User  `gorm:"many2many:users_grants"`

	Kubernetes GrantKubernetes
}

type GrantKubernetesKind string

var (
	GrantKubernetesKindGrant        GrantKubernetesKind = "grant"
	GrantKubernetesKindClusterGrant GrantKubernetesKind = "cluster-grant"
)

type GrantKubernetes struct {
	Model

	Kind      GrantKubernetesKind
	Name      string
	Namespace string

	GrantID uuid.UUID
}

func (r *Grant) ToAPI() api.Grant {
	result := api.Grant{
		ID:      r.ID.String(),
		Created: r.CreatedAt.Unix(),
		Updated: r.UpdatedAt.Unix(),
	}

	switch r.Kind {
	case GrantKindKubernetes:
		result.Kind = api.GrantKind(r.Kubernetes.Kind)
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
