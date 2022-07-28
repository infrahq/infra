package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

func validateProviderUser(u *models.ProviderUser) error {
	switch {
	case u.ProviderID == 0:
		return fmt.Errorf("providerID is required")
	case u.IdentityID == 0:
		return fmt.Errorf("identityID is required")
	case u.Email == "":
		return fmt.Errorf("email is required")
	case u.LastUpdate.IsZero():
		return fmt.Errorf("lastUpdated is required")
	default:
		return nil
	}
}

func CreateProviderUser(db *gorm.DB, provider *models.Provider, ident *models.Identity) (*models.ProviderUser, error) {
	pu, err := get[models.ProviderUser](db, ByIdentityID(ident.ID), ByProviderID(provider.ID))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return nil, err
	}

	if pu != nil {
		return pu, nil
	}

	pu = &models.ProviderUser{
		ProviderID: provider.ID,
		IdentityID: ident.ID,
		Email:      ident.Name,
		LastUpdate: time.Now().UTC(),
	}
	if err := validateProviderUser(pu); err != nil {
		return nil, err
	}
	return pu, add(db, pu)
}

func UpdateProviderUser(db *gorm.DB, providerUser *models.ProviderUser) error {
	if err := validateProviderUser(providerUser); err != nil {
		return err
	}
	return save(db, providerUser)
}

func ListProviderUsers(db *gorm.DB, p *models.Pagination, selectors ...SelectorFunc) ([]models.ProviderUser, error) {
	return list[models.ProviderUser](db, p, selectors...)
}

func DeleteProviderUsers(db *gorm.DB, selectors ...SelectorFunc) error {
	return deleteAll[models.ProviderUser](db, selectors...)
}

func GetProviderUser(db *gorm.DB, providerID, userID uid.ID) (*models.ProviderUser, error) {
	return get[models.ProviderUser](db, ByProviderID(providerID), ByIdentityID(userID))
}

func SyncProviderUser(ctx context.Context, db *gorm.DB, user *models.Identity, provider *models.Provider, oidcClient providers.OIDCClient) error {
	providerUser, err := GetProviderUser(db, provider.ID, user.ID)
	if err != nil {
		return err
	}

	accessToken, expiry, err := oidcClient.RefreshAccessToken(ctx, providerUser)
	if err != nil {
		return fmt.Errorf("refresh provider access: %w", err)
	}

	// update the stored access token if it was refreshed
	if accessToken != string(providerUser.AccessToken) {
		logging.Debugf("access token for user at provider %s was refreshed", providerUser.ProviderID)

		providerUser.AccessToken = models.EncryptedAtRest(accessToken)
		providerUser.ExpiresAt = *expiry

		err = UpdateProviderUser(db, providerUser)
		if err != nil {
			return fmt.Errorf("update provider user on sync: %w", err)
		}
	}

	info, err := oidcClient.GetUserInfo(ctx, providerUser)
	if err != nil {
		return fmt.Errorf("oidc user sync failed: %w", err)
	}

	if err := AssignIdentityToGroups(db, user, provider, info.Groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}
