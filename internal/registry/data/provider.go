package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

func SetProviderUsers(db *gorm.DB, provider *models.Provider, emails ...string) error {
	users, err := ListUsers(db, db.Where("email IN (?)", emails))
	if err != nil {
		return err
	}

	if err := db.Model(provider).Association("Users").Replace(users); err != nil {
		return err
	}

	return nil
}

func SetProviderGroups(db *gorm.DB, provider *models.Provider, names ...string) error {
	groups, err := ListGroups(db, db.Where("name IN (?)", names))
	if err != nil {
		return err
	}

	if err := db.Model(provider).Association("Groups").Replace(groups); err != nil {
		return err
	}

	return nil
}

func CreateProvider(db *gorm.DB, provider *models.Provider) (*models.Provider, error) {
	if err := add(db, &models.Provider{}, provider, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func CreateOrUpdateProvider(db *gorm.DB, provider *models.Provider, condition interface{}) (*models.Provider, error) {
	existing, err := GetProvider(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateProvider(db, provider); err != nil {
			return nil, err
		}

		return provider, nil
	}

	if err := update(db, &models.Provider{}, provider, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch provider.Kind {
	case models.ProviderKindOkta:
		if err := db.Model(existing).Association("Okta").Replace(&provider.Okta); err != nil {
			return nil, err
		}
	}

	return GetProvider(db, db.Where(existing, "id"))
}

func GetProvider(db *gorm.DB, condition interface{}) (*models.Provider, error) {
	var provider models.Provider
	if err := get(db, &models.Provider{}, &provider, condition); err != nil {
		return nil, err
	}

	return &provider, nil
}

func ListProviders(db *gorm.DB, condition interface{}) ([]models.Provider, error) {
	providers := make([]models.Provider, 0)
	if err := list(db, &models.Provider{}, &providers, condition); err != nil {
		return nil, err
	}

	return providers, nil
}

func UpdateProvider(db *gorm.DB, provider *models.Provider, selector SelectorFunc) (*models.Provider, error) {
	existing, err := GetProvider(db, selector(db))
	if err != nil {
		return nil, err
	}

	if err := update(db, &models.Provider{}, provider, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch provider.Kind {
	case models.ProviderKindOkta:
		if err := db.Model(existing).Association("Okta").Replace(&provider.Okta); err != nil {
			return nil, err
		}
	}

	return GetProvider(db, db.Where(existing, "id"))
}

func DeleteProviders(db *gorm.DB, selector SelectorFunc) error {
	toDelete, err := ListProviders(db, selector(db))
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return delete(db, &models.Provider{}, ids)
	}

	return internal.ErrNotFound
}
