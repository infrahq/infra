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

		providerUsers, err := ListProviderUsers(db, ByProviderID(p.ID))
		if err != nil {
			return err
		}

		// if a user has no other providers, we need to remove the user.
		userIDsToDelete := []uid.ID{}
		for _, providerUser := range providerUsers {
			user, err := GetIdentity(db.Preload("Providers"), ByID(providerUser.IdentityID))
			if err != nil {
				return err
			}

			if len(user.Providers) == 1 && user.Providers[0].ID == p.ID {
				userIDsToDelete = append(userIDsToDelete, user.ID)
			}
		}

		if len(userIDsToDelete) > 0 {
			if err := DeleteIdentities(db, ByIDs(userIDsToDelete)); err != nil {
				return err
			}
		}

		if err := DeleteProviderUsers(db, ByProviderID(p.ID)); err != nil {
			return err
		}

		if err := DeleteAccessKeys(db, ByProviderID(p.ID)); err != nil {
			return err
		}
	}

	return deleteAll[models.Provider](db, ByIDs(ids))
}
