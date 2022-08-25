package data

import (
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func validateProvider(p *models.Provider) error {
	switch {
	case p.Name == "":
		return fmt.Errorf("name is required")
	case p.Kind == "":
		return fmt.Errorf("kind is required")
	default:
		return nil
	}
}

func CreateProvider(db GormTxn, provider *models.Provider) error {
	if err := validateProvider(provider); err != nil {
		return err
	}
	return add(db, provider)
}

func GetProvider(db GormTxn, selectors ...SelectorFunc) (*models.Provider, error) {
	return get[models.Provider](db, selectors...)
}

func ListProviders(db GormTxn, p *models.Pagination, selectors ...SelectorFunc) ([]models.Provider, error) {
	return list[models.Provider](db, p, selectors...)
}

func SaveProvider(db GormTxn, provider *models.Provider) error {
	if err := validateProvider(provider); err != nil {
		return err
	}
	return save(db, provider)
}

func DeleteProviders(db GormTxn, selectors ...SelectorFunc) error {
	toDelete, err := ListProviders(db, nil, selectors...)
	if err != nil {
		return fmt.Errorf("listing providers: %w", err)
	}

	ids := make([]uid.ID, 0)
	for _, p := range toDelete {
		ids = append(ids, p.ID)

		providerUsers, err := ListProviderUsers(db, nil, ByProviderID(p.ID))
		if err != nil {
			return fmt.Errorf("listing provider users: %w", err)
		}

		// if a user has no other providers, we need to remove the user.
		userIDsToDelete := []uid.ID{}
		for _, providerUser := range providerUsers {
			user, err := GetIdentity(db, Preload("Providers"), ByID(providerUser.IdentityID))
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

type providersCount struct {
	Kind  string
	Count float64
}

func CountProvidersByKind(tx GormTxn) ([]providersCount, error) {
	db := tx.GormDB()
	var results []providersCount
	if err := db.Raw("SELECT kind, COUNT(*) as count FROM providers WHERE kind <> 'infra' AND deleted_at IS NULL GROUP BY kind").Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
