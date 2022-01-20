package models

import (
	"github.com/infrahq/infra/uuid"

	"github.com/infrahq/infra/internal/api"
)

type GrantKind string

var (
	GrantKindInfra      GrantKind = "infra"
	GrantKindKubernetes GrantKind = "kubernetes"
)

type Grant struct {
	Model
	Kind GrantKind `validate:"required"`

	DestinationID uuid.UUID `validate:"required"`
	Destination   *Destination

	Groups []Group `gorm:"many2many:groups_grants"`
	Users  []User  `gorm:"many2many:users_grants"`

	Kubernetes GrantKubernetes
}

type GrantKubernetesKind string

var (
	GrantKubernetesKindRole        GrantKubernetesKind = "role"
	GrantKubernetesKindClusterRole GrantKubernetesKind = "cluster-role"
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
		Kind:    api.GrantKind(r.Kind),
	}

	switch r.Kind {
	case GrantKindKubernetes:
		result.Kubernetes = &api.GrantKubernetes{
			Kind:      api.GrantKubernetesKind(r.Kubernetes.Kind),
			Name:      r.Kubernetes.Name,
			Namespace: r.Kubernetes.Namespace,
		}
	case GrantKindInfra:
	}

	users := make([]api.User, 0)
	for _, u := range r.Users {
		users = append(users, u.ToAPI())
	}

	if len(users) > 0 {
		result.Users = users
	}

	groups := make([]api.Group, 0)
	for _, g := range r.Groups {
		groups = append(groups, g.ToAPI())
	}

	if len(groups) > 0 {
		result.Groups = groups
	}

	result.Destination = r.Destination.ToAPI()

	return result
}
