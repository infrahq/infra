package data

import (
	"errors"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
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

		err := DeleteProviderUsers(db, ByProviderID(p.ID))
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

var infraProviderCache *models.Provider

// InfraProvider is a lazy-loaded cached reference to the infra provider, since it's used in a lot of places
func InfraProvider(db *gorm.DB) *models.Provider {
	if infraProviderCache != nil {
		return infraProviderCache
	}

	infra, err := get[models.Provider](db, ByName("infra"))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			logging.S.Panic(err)
			return nil
		}

		// create the infra provider since it doesn't exist.
		infra = &models.Provider{Name: "infra"}
		err = add(db, infra)
		if err != nil {
			logging.S.Error(err)
			return nil
		}
	}

	infraProviderCache = infra
	return infra
}
