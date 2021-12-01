package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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
		Id:      r.ID.String(),
		Created: r.CreatedAt.Unix(),
		Updated: r.UpdatedAt.Unix(),
	}

	switch r.Kind {
	case RoleKindKubernetes:
		result.Kind = api.RoleKind(r.Kubernetes.Kind)
		result.Name = r.Kubernetes.Name
		result.Namespace = r.Kubernetes.Namespace
	}

	for _, u := range r.Users {
		result.Users = append(result.Users, u.ToAPI())
	}

	for _, g := range r.Groups {
		result.Groups = append(result.Groups, g.ToAPI())
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

func CreateRole(db *gorm.DB, role *Role) (*Role, error) {
	if err := add(db, &Role{}, role, &Role{}); err != nil {
		return nil, err
	}

	return role, nil
}

func CreateOrUpdateRole(db *gorm.DB, role *Role, condition interface{}) (*Role, error) {
	existing, err := GetRole(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateRole(db, role); err != nil {
			return nil, err
		}

		return role, nil
	}

	if err := update(db, &Role{}, role, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch role.Kind {
	case RoleKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&role.Kubernetes); err != nil {
			return nil, err
		}
	}

	return GetRole(db, db.Where(existing, "id"))
}

func GetRole(db *gorm.DB, condition interface{}) (*Role, error) {
	var role Role
	if err := get(db, &Role{}, &role, condition); err != nil {
		return nil, err
	}

	return &role, nil
}

func ListRoles(db *gorm.DB, condition interface{}) ([]Role, error) {
	roles := make([]Role, 0)
	if err := list(db, &Role{}, &roles, condition); err != nil {
		return nil, err
	}

	return roles, nil
}

func DeleteRoles(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListRoles(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &Role{}, ids)
	}

	return nil
}

func RoleSelector(db *gorm.DB, role *Role) *gorm.DB {
	switch role.Kind {
	case RoleKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&RoleKubernetes{}).Select("role_id").Where(role.Kubernetes),
		)
	}

	if role.Destination.ID != uuid.Nil {
		db = db.Where("destination_id in (?)", role.Destination.ID)
	}

	return db.Where(&role)
}

// StrictRoleSelector matches all fields exactly, including initialized fields.
func StrictRoleSelector(db *gorm.DB, role *Role) *gorm.DB {
	switch role.Kind {
	case RoleKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&RoleKubernetes{}).Select("role_id").Where(map[string]interface{}{
				"Kind":      role.Kubernetes.Kind,
				"Name":      role.Kubernetes.Name,
				"Namespace": role.Kubernetes.Namespace,
			}),
		)
	}

	if role.Destination.ID != uuid.Nil {
		db = db.Where("destination_id in (?)", role.Destination.ID)
	}

	return db.Where(&role)
}
