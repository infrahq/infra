package data

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateProvider(db *gorm.DB, provider *models.Provider) error {
	return add(db, provider)
}

func GetProvider(db *gorm.DB, selectors ...SelectorFunc) (*models.Provider, error) {
	return get[models.Provider](db, selectors...)
}

func ListProviders(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.Provider, error) {
	return list[models.Provider](db, p, selectors...)
}

func SaveProvider(db *gorm.DB, provider *models.Provider) error {
	return save(db, provider)
}

func DeleteProviders(db *gorm.DB, selectors ...SelectorFunc) error {
	// Better solution here needed when Pagination becomes mandatory, Replace with a multiple-page "ListAll" function?
	toDelete, err := ListProviders(db, &models.Pagination{}, selectors...)
	if err != nil {
		return fmt.Errorf("listing providers: %w", err)
	}

	ids := make([]uid.ID, 0)
	for _, p := range toDelete {
		ids = append(ids, p.ID)

		// Same as toDelete
		providerUsers, err := ListProviderUsers(db, &models.Pagination{}, ByProviderID(p.ID))
		if err != nil {
			return fmt.Errorf("listing provider users: %w", err)
		}

		// if a user has no other providers, we need to remove the user.
		userIDsToDelete := []uid.ID{}
		for _, providerUser := range providerUsers {
			user, err := GetIdentity(db.Preload("Providers"), ByID(providerUser.IdentityID))
			if err != nil {
				if errors.Is(err, internal.ErrNotFound) {
					continue
				}
				return fmt.Errorf("get user: %w", err)
			}

			if len(user.Providers) == 1 && user.Providers[0].ID == p.ID {
				userIDsToDelete = append(userIDsToDelete, user.ID)
			}
		}

		if len(userIDsToDelete) > 0 {
			if err := DeleteIdentities(db, ByIDs(userIDsToDelete)); err != nil {
				return fmt.Errorf("delete users: %w", err)
			}
		}

		if err := DeleteProviderUsers(db, ByProviderID(p.ID)); err != nil {
			return fmt.Errorf("delete provider users: %w", err)
		}

		if err := DeleteAccessKeys(db, ByProviderID(p.ID)); err != nil {
			return fmt.Errorf("delete access keys: %w", err)
		}
	}

	return deleteAll[models.Provider](db, ByIDs(ids))
}
