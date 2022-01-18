package data

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

func AppendProviderUsers(db *gorm.DB, provider *models.Provider, user *models.User) error {
	if err := db.Model(provider).Association("Users").Append(user); err != nil {
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

func CreateProvider(db *gorm.DB, provider *models.Provider) error {
	return add(db, provider)
}

func GetProvider(db *gorm.DB, selectors ...SelectorFunc) (*models.Provider, error) {
	return get[models.Provider](db, selectors...)
}

func ListProviders(db *gorm.DB, selectors ...SelectorFunc) ([]models.Provider, error) {
	return list[models.Provider](db, selectors...)
}

func SaveProvider(db *gorm.DB, provider *models.Provider) error {
	return save(db, provider)
}

func DeleteProviders(db *gorm.DB, selectors ...SelectorFunc) error {
	toDelete, err := ListProviders(db, selectors...)
	if err != nil {
		return err
	}

	ids := make([]uid.ID, 0)
	for _, p := range toDelete {
		ids = append(ids, p.ID)

		err := DeleteUsers(db, ByProviderID(p.ID))
		if err != nil {
			return err
		}

		err = DeleteGroups(db, ByProviderID(p.ID))
		if err != nil {
			return err
		}
	}

	return deleteAll[models.Provider](db, ByIDs(ids))
}

func CreateProviderToken(db *gorm.DB, token *models.ProviderToken) error {
	return add(db, token)
}

func UpdateProviderToken(db *gorm.DB, token *models.ProviderToken) error {
	return save(db, token)
}

func GetProviderToken(db *gorm.DB, selector SelectorFunc) (*models.ProviderToken, error) {
	return get[models.ProviderToken](db, selector)
}
