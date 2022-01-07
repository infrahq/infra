package data

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func CreateGrant(db *gorm.DB, grant *models.Grant) (*models.Grant, error) {
	if err := add(db, &models.Grant{}, grant, &models.Grant{}); err != nil {
		return nil, err
	}

	return grant, nil
}

func CreateOrUpdateGrant(db *gorm.DB, grant *models.Grant) (*models.Grant, error) {
	existing, err := GetGrantByModel(db, grant)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, fmt.Errorf("get: %w", err)
		}

		if _, err := CreateGrant(db, grant); err != nil {
			return nil, fmt.Errorf("create: %w", err)
		}

		return grant, nil
	}

	if err := update(db, &models.Grant{}, grant, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch grant.Kind {
	case models.GrantKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&grant.Kubernetes); err != nil {
			return nil, fmt.Errorf("update: %w", err)
		}
	}

	return GetGrant(db, db.Where(existing, "id"))
}

func GetGrant(db *gorm.DB, condition interface{}) (*models.Grant, error) {
	var grant models.Grant
	if err := get(db, &models.Grant{}, &grant, condition); err != nil {
		return nil, err
	}

	return &grant, nil
}

func ListUserGrants(db *gorm.DB, userID uuid.UUID) (result []models.Grant, err error) {
	err = db.Model((*models.Grant)(nil)).Where("user_id = ?", userID).Find(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetGrantByModel(db *gorm.DB, grant *models.Grant) (result *models.Grant, err error) {
	result = &models.Grant{}

	err = db.Model(&models.Grant{}).
		Joins("left join grant_kubernetes g on g.grant_id = grants.id").
		Where("grants.destination_id = ? and grants.kind = ? and g.namespace = ? and g.name = ?", grant.DestinationID, grant.Kind, grant.Kubernetes.Namespace, grant.Kubernetes.Name).
		First(result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, internal.ErrNotFound
		}

		return nil, err
	}

	return result, nil
}

func ListGrants(db *gorm.DB, selectors ...SelectorFunc) ([]models.Grant, error) {
	grants := make([]models.Grant, 0)
	if err := list(db, &models.Grant{}, &grants, selectors); err != nil {
		return nil, err
	}

	return grants, nil
}

func DeleteGrants(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListGrants(db, selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &models.Grant{}, ids)
	}

	return internal.ErrNotFound
}

// StrictGrantSelector matches all fields exactly, including initialized fields.
func StrictGrantSelector(db *gorm.DB, grant *models.Grant) *gorm.DB {
	return db.Joins("left join grant_kubernetes g on g.grant_id = grants.id").Where("grants.destination_id = ? and grants.kind = ? and grant_kubernetes.namespace = ? and grant_kubernetes.name = ?", grant.Destination.ID, grant.Kind, grant.Kubernetes.Namespace, grant.Kubernetes.Name)
}

func ByGrantKind(kind models.GrantKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(kind) == 0 {
			return db
		}

		switch kind {
		case models.GrantKindKubernetes:
			return db.Where("kind = ?", kind)
		default:
			// panic("unknown grant kind: " + string(kind))
			db.Logger.Error(db.Statement.Context, "unknown grant kind: "+string(kind))
			db.AddError(fmt.Errorf("%w: unknown grant kind: %q", internal.ErrBadRequest, string(kind)))
			return db.Where("1 = 2")
		}
	}
}

func ByDestinationKind(kind models.DestinationKind) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(kind) == 0 {
			return db
		}

		switch kind {
		case models.DestinationKindKubernetes:
			return db.Where("kind = ?", kind)
		default:
			db.Logger.Error(db.Statement.Context, "unknown destination kind: "+string(kind))
			db.AddError(fmt.Errorf("%w: unknown destination kind: %q", internal.ErrBadRequest, string(kind)))
			return db.Where("1 = 2")
		}
	}
}

func NotByIDs(ids []uuid.UUID) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		if len(ids) == 0 {
			return db
		}

		return db.Where("id not in (?)", ids)
	}
}
