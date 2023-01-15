package access

import (
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetProviderUser(rCtx RequestContext, id uid.ID) (*models.ProviderUser, error) {
	// IssuedFor will match no providers if called with a regular access key. When called with
	// a SCIM access key it will be the provider ID. This effectively restricts this endpoint to
	// only SCIM access keys.
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return nil, err
	}
	user, err := data.GetProviderUser(rCtx.DBTxn, rCtx.Authenticated.AccessKey.IssuedFor, id)
	if err != nil {
		return nil, fmt.Errorf("get provider users: %w", err)
	}
	return user, nil
}

func ListProviderUsers(rCtx RequestContext, p *data.SCIMParameters) ([]models.ProviderUser, error) {
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return []models.ProviderUser{}, err
	}
	opts := data.ListProviderUsersOptions{
		ByProviderID:   rCtx.Authenticated.AccessKey.IssuedFor,
		SCIMParameters: p,
	}
	users, err := data.ListProviderUsers(rCtx.DBTxn, opts)
	if err != nil {
		return nil, fmt.Errorf("list provider users: %w", err)
	}
	return users, nil
}

func CreateProviderUser(rCtx RequestContext, u *models.ProviderUser) error {
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return err
	}
	u.ProviderID = rCtx.Authenticated.AccessKey.IssuedFor
	err := data.ProvisionProviderUser(rCtx.DBTxn, u)
	if err != nil {
		return fmt.Errorf("provision provider user: %w", err)
	}
	return nil
}

func UpdateProviderUser(rCtx RequestContext, u *models.ProviderUser) error {
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return err
	}
	u.ProviderID = rCtx.Authenticated.AccessKey.IssuedFor
	err := data.UpdateProviderUser(rCtx.DBTxn, u)
	if err != nil {
		if errors.Is(err, data.ErrSourceOfTruthConflict) {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
		return fmt.Errorf("update provider user: %w", err)
	}
	return nil
}

func PatchProviderUser(rCtx RequestContext, u *models.ProviderUser) (*models.ProviderUser, error) {
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return nil, err
	}
	u.ProviderID = rCtx.Authenticated.AccessKey.IssuedFor
	updated, err := data.PatchProviderUserActiveStatus(rCtx.DBTxn, u)
	if err != nil {
		return nil, fmt.Errorf("patch provider user: %w", err)
	}
	return updated, nil
}

func DeleteProviderUser(rCtx RequestContext, userID uid.ID) error {
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(rCtx); err != nil {
		return err
	}
	providerID := rCtx.Authenticated.AccessKey.IssuedFor
	// delete the provider user, and if its the last reference to the user, remove their identity also
	opts := data.DeleteIdentitiesOptions{
		ByProviderID: providerID,
		ByIDs:        []uid.ID{userID},
	}
	if err := data.DeleteIdentities(rCtx.DBTxn, opts); err != nil {
		return fmt.Errorf("delete provider user identity: %w", err)
	}
	return nil
}

func checkKeyIdentityProvider(rCtx RequestContext) error {
	_, err := data.GetProvider(rCtx.DBTxn,
		data.GetProviderOptions{ByID: rCtx.Authenticated.AccessKey.IssuedFor})
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return internal.ErrUnauthorized
		}
		return fmt.Errorf("validate scim provider: %w", err)
	}
	return nil
}
