package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetProviderUser(c *gin.Context, id uid.ID) (*models.ProviderUser, error) {
	// IssuedFor will match no providers if called with a regular access key. When called with
	// a SCIM access key it will be the provider ID. This effectively restricts this endpoint to
	// only SCIM access keys.
	ctx := GetRequestContext(c)
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return nil, err
	}
	user, err := data.GetProviderUser(ctx.DBTxn, ctx.Authenticated.AccessKey.IssuedFor, id)
	if err != nil {
		return nil, fmt.Errorf("get provider users: %w", err)
	}
	return user, nil
}

func ListProviderUsers(c *gin.Context, p *data.SCIMParameters) ([]models.ProviderUser, error) {
	ctx := GetRequestContext(c)
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return []models.ProviderUser{}, err
	}
	opts := data.ListProviderUsersOptions{
		ByProviderID:   ctx.Authenticated.AccessKey.IssuedFor,
		SCIMParameters: p,
	}
	users, err := data.ListProviderUsers(ctx.DBTxn, opts)
	if err != nil {
		return nil, fmt.Errorf("list provider users: %w", err)
	}
	return users, nil
}

func CreateProviderUser(c *gin.Context, u *models.ProviderUser) error {
	ctx := GetRequestContext(c)
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return err
	}
	u.ProviderID = ctx.Authenticated.AccessKey.IssuedFor
	err := data.ProvisionProviderUser(ctx.DBTxn, u)
	if err != nil {
		return fmt.Errorf("provision provider user: %w", err)
	}
	return nil
}

func UpdateProviderUser(c *gin.Context, u *models.ProviderUser) error {
	ctx := GetRequestContext(c)
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return err
	}
	u.ProviderID = ctx.Authenticated.AccessKey.IssuedFor
	err := data.UpdateProviderUser(ctx.DBTxn, u)
	if err != nil {
		if errors.Is(err, data.ErrSourceOfTruthConflict) {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
		return fmt.Errorf("update provider user: %w", err)
	}
	return nil
}

func PatchProviderUser(c *gin.Context, u *models.ProviderUser) (*models.ProviderUser, error) {
	ctx := GetRequestContext(c)
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return nil, err
	}
	u.ProviderID = ctx.Authenticated.AccessKey.IssuedFor
	updated, err := data.PatchProviderUserActiveStatus(ctx.DBTxn, u)
	if err != nil {
		return nil, fmt.Errorf("patch provider user: %w", err)
	}
	return updated, nil
}

func DeleteProviderUser(c *gin.Context, userID uid.ID) error {
	ctx := GetRequestContext(c)
	// restricted to only SCIM access keys
	if err := checkKeyIdentityProvider(ctx); err != nil {
		return err
	}
	providerID := ctx.Authenticated.AccessKey.IssuedFor
	// delete the provider user, and if its the last reference to the user, remove their identity also
	opts := data.DeleteIdentitiesOptions{
		ByProviderID: providerID,
		ByIDs:        []uid.ID{userID},
	}
	if err := data.DeleteIdentities(ctx.DBTxn, opts); err != nil {
		return fmt.Errorf("delete provider user identity: %w", err)
	}
	return nil
}

func checkKeyIdentityProvider(ctx RequestContext) error {
	_, err := data.GetProvider(ctx.DBTxn,
		data.GetProviderOptions{ByID: ctx.Authenticated.AccessKey.IssuedFor})
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return internal.ErrUnauthorized
		}
		return fmt.Errorf("validate scim provider: %w", err)
	}
	return nil
}
