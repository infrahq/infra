package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateRole(db *gorm.DB, role *models.Role) (*models.Role, error) {
	if err := add(db, &models.Role{}, role, &models.Role{}); err != nil {
		return nil, err
	}

	return role, nil
}

func CreateOrUpdateRole(db *gorm.DB, role *models.Role, condition interface{}) (*models.Role, error) {
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

	if err := update(db, &models.Role{}, role, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch role.Kind {
	case models.RoleKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&role.Kubernetes); err != nil {
			return nil, err
		}
	}

	return GetRole(db, db.Where(existing, "id"))
}

func GetRole(db *gorm.DB, condition interface{}) (*models.Role, error) {
	var role models.Role
	if err := get(db, &models.Role{}, &role, condition); err != nil {
		return nil, err
	}

	return &role, nil
}

func ListRoles(db *gorm.DB, condition interface{}) ([]models.Role, error) {
	roles := make([]models.Role, 0)
	if err := list(db, &models.Role{}, &roles, condition); err != nil {
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

		return remove(db, &models.Role{}, ids)
	}

	return nil
}

func RoleSelector(db *gorm.DB, role *models.Role) *gorm.DB {
	switch role.Kind {
	case models.RoleKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&models.RoleKubernetes{}).Select("role_id").Where(role.Kubernetes),
		)
	}

	if role.Destination.ID != uuid.Nil {
		db = db.Where("destination_id in (?)", role.Destination.ID)
	}

	return db.Where(&role)
}

// StrictRoleSelector matches all fields exactly, including initialized fields.
func StrictRoleSelector(db *gorm.DB, role *models.Role) *gorm.DB {
	switch role.Kind {
	case models.RoleKindKubernetes:
		db = db.Where(
			"id IN (?)",
			db.Model(&models.RoleKubernetes{}).Select("role_id").Where(map[string]interface{}{
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
