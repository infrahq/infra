package data

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

var mu sync.Mutex

func AppendProviderUsers(db *gorm.DB, provider *models.Provider, user models.User) error {
	mu.Lock()
	defer mu.Unlock()

	users := provider.Users
	users = append(users, user)

	if err := db.Model(provider).Association("Users").Replace(users); err != nil {
		return fmt.Errorf("append provider users: %w", err)
	}

	return nil
}

func AppendProviderGroups(db *gorm.DB, provider *models.Provider, group *models.Group) error {
	if err := db.Model(provider).Association("Groups").Append(group); err != nil {
		return fmt.Errorf("append provider groups: %w", err)
	}

	return nil
}

func CreateProvider(db *gorm.DB, provider *models.Provider) (*models.Provider, error) {
	if err := add(db, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// CreateOrUpdateProvider is deprecated
func CreateOrUpdateProvider(db *gorm.DB, provider *models.Provider) (*models.Provider, error) {
	existing, err := GetProvider(db, ByProviderKind(provider.Kind), ByDomain(provider.Domain))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateProvider(db, provider); err != nil {
			return nil, err
		}

		return provider, nil
	}

	if err := update(db, existing.ID, provider); err != nil {
		return nil, err
	}

	return get[models.Provider](db, ByID(existing.ID))
}

func GetProvider(db *gorm.DB, selectors ...SelectorFunc) (*models.Provider, error) {
	return get[models.Provider](db, selectors...)
}

func ListProviders(db *gorm.DB, selectors ...SelectorFunc) ([]models.Provider, error) {
	return list[models.Provider](db, selectors...)
}

func UpdateProvider(db *gorm.DB, provider *models.Provider, selector SelectorFunc) (*models.Provider, error) {
	existing, err := GetProvider(db, selector)
	if err != nil {
		return nil, err
	}

	if err := save(db, existing); err != nil {
		return nil, err
	}

	return GetProvider(db, ByID(existing.ID))
}

func DeleteProviders(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListProviders(db, selectors...)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uid.ID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return removeAll[models.Provider](db, ByIDs(ids))
	}

	return internal.ErrNotFound
}
