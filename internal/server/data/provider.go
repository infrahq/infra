package data

import (
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

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

		err := DeleteIdentities(db, ByProviderID(p.ID))
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

func AppendProviderUsers(db *gorm.DB, provider *models.Provider, user *models.Identity) error {
	return appendAssociation(db, provider, "Users", user)
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
