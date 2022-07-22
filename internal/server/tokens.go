package server

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateToken(c *gin.Context, _ *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	rCtx := getRequestContext(c)
	if rCtx.Authenticated.User == nil {
		return nil, fmt.Errorf("%w: no identity found in access key", internal.ErrUnauthorized)
	}

	if err := updateIdentityInfoFromProvider(rCtx); err != nil {
		// this will fail if the user was removed from the IDP, which means they no longer are a valid user
		return nil, fmt.Errorf("%w: failed to update identity info from provider: %s", internal.ErrUnauthorized, err)
	}

	token, err := data.CreateIdentityToken(rCtx.DBTxn, rCtx.Authenticated.User.ID)
	if err != nil {
		return nil, err
	}

	return &api.CreateTokenResponse{Token: token.Token, Expires: api.Time(token.Expires)}, nil
}

// updateIdentityInfoFromProvider calls the identity provider used to authenticate
// this user session to update their current information.
func updateIdentityInfoFromProvider(rCtx RequestContext) error {
	db := rCtx.DBTxn

	providerUser, err := data.GetProviderUser(
		db,
		rCtx.Authenticated.AccessKey.ProviderID,
		rCtx.Authenticated.User.ID)
	if err != nil {
		return err
	}

	provider, err := data.GetProvider(db, data.ByID(providerUser.ProviderID))
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	if provider.Name == models.InternalInfraProviderName || provider.Kind == models.ProviderKindInfra {
		return nil
	}

	oidc, err := newProviderOIDCClient(rCtx, provider, providerUser.RedirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	ctx := rCtx.Request.Context()
	user := rCtx.Authenticated.User
	// get current identity provider groups and account status
	err = data.SyncProviderUser(ctx, db, user, provider, oidc)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			return err
		}

		if nestedErr := data.DeleteAccessKeys(db, data.ByIssuedFor(user.ID)); nestedErr != nil {
			logging.Errorf("failed to revoke invalid user session: %s", nestedErr)
		}

		if nestedErr := data.DeleteProviderUsers(db, data.ByIdentityID(user.ID), data.ByProviderID(provider.ID)); nestedErr != nil {
			logging.Errorf("failed to delete provider user: %s", nestedErr)
		}

		return fmt.Errorf("sync user: %w", err)
	}

	return nil
}
