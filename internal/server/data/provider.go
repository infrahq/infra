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

func GetProvider(db *gorm.DB, opt IDOrNameQuery) (*models.Provider, error) {
	q := Query(`SELECT * from providers WHERE`)
	switch {
	case opt.ID != 0:
		q.B(`id = $1`, opt.ID)
	case opt.Name != "":
		q.B(`name = $1`, opt.Name)
	default:
		return nil, fmt.Errorf("query requires either an ID or Name")
	}
	q.B("AND deleted_at is null")
	q.B("LIMIT 1")

	var p models.Provider
	result := db.Raw(q.String(), q.Args...).First(&p)
	return &p, handleReadError(result.Error)
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

		if err := DeleteAccessKeys(db, DeleteAccessKeysQuery{ProviderID: p.ID}); err != nil {
			return fmt.Errorf("delete access keys: %w", err)
		}
	}

	return deleteAll[models.Provider](db, ByIDs(ids))
}

type providersCount struct {
	Kind  string
	Count float64
}

func CountProvidersByKind(db *gorm.DB) ([]providersCount, error) {
	var results []providersCount
	if err := db.Raw("SELECT kind, COUNT(*) as count FROM providers WHERE deleted_at IS NULL GROUP BY kind").Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
