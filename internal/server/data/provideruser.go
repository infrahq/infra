package data

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateProviderUser(db *gorm.DB, provider *models.Provider, ident *models.Identity) (*models.ProviderUser, error) {
	pu, err := get[models.ProviderUser](db, ByIdentityID(ident.ID), ByProviderID(provider.ID))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return nil, err
	}

	if pu == nil {
		pu = &models.ProviderUser{
			ProviderID: provider.ID,
			IdentityID: ident.ID,
			Email:      ident.Name,
			LastUpdate: time.Now().UTC(),
		}
		if err := add(db, pu); err != nil {
			return nil, err
		}
	}

	// If there were other attributes to udpate, I guess they should be updated here.

	return pu, nil
}

func UpdateProviderUser(db *gorm.DB, providerUser *models.ProviderUser) error {
	return save(db, providerUser)
}

func DeleteProviderUsers(db *gorm.DB, selector SelectorFunc) error {
	return deleteAll[models.ProviderUser](db, selector)
}

func GetProviderUser(db *gorm.DB, providerID, userID uid.ID) (*models.ProviderUser, error) {
	return get[models.ProviderUser](db, ByProviderID(providerID), ByIdentityID(userID))
}
