package data

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func BindUserGrants(db *gorm.DB, user *models.User, grantIDs ...uuid.UUID) error {
	grants, err := ListGrants(db, grantIDs)
	if err != nil {
		return err
	}

	if err := db.Model(user).Association("Grants").Replace(grants); err != nil {
		return err
	}

	return nil
}

func CreateUser(db *gorm.DB, user *models.User) (*models.User, error) {
	if err := add(db, &models.User{}, user, &models.User{}); err != nil {
		return nil, err
	}

	return user, nil
}

func CreateOrUpdateUser(db *gorm.DB, user *models.User, condition interface{}) (*models.User, error) {
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

	return UpdateUser(db, user, ByUUID(existing.ID))
}

func GetUser(db *gorm.DB, condition interface{}) (*models.User, error) {
	var user models.User
	if err := get(db, &models.User{}, &user, condition); err != nil {
		return nil, err
	}

	return &user, nil
}

func ListUsers(db *gorm.DB, condition interface{}) ([]models.User, error) {
	users := make([]models.User, 0)
	if err := list(db, &models.User{}, &users, condition); err != nil {
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

		return remove(db, &models.User{}, ids)
	}

	return nil
}

func UpdateUser(db *gorm.DB, user *models.User, selector SelectorFunc) (*models.User, error) {
	existing, err := GetUser(db, selector(db))
	if err != nil {
		return nil, fmt.Errorf("get existing: %w", err)
	}

	if err := update(db, &models.User{}, user, db.Where(existing, "id")); err != nil {
		return nil, fmt.Errorf("update: %w", err)
	}

	if err := db.Model(existing).Association("Grants").Replace(&user.Grants); err != nil {
		return nil, fmt.Errorf("grants: %w", err)
	}

	if err := db.Model(existing).Association("Providers").Replace(&user.Providers); err != nil {
		return nil, fmt.Errorf("providers: %w", err)
	}

	if err := db.Model(existing).Association("Groups").Replace(&user.Groups); err != nil {
		return nil, fmt.Errorf("groups: %w", err)
	}

	return GetUser(db, db.Where(existing, "id"))
}

func UserAssociations(db *gorm.DB) *gorm.DB {
	db = db.Preload("Grants.Kubernetes").Preload("Grants.Destination.Kubernetes")
	db = db.Preload("Groups.Grants.Kubernetes").Preload("Groups.Grants.Destination.Kubernetes")

	return db
}
