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
	"github.com/infrahq/infra/internal/server/providers"
)

// TODO: remove method receiver
func (a *API) CreateToken(c *gin.Context, _ *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	rCtx := getRequestContext(c)
	if rCtx.Authenticated.User == nil {
		return nil, fmt.Errorf("%w: no identity found in access key", internal.ErrUnauthorized)
	}

	err := a.updateIdentityInfoFromProvider(rCtx)
	if err != nil {
		// this will fail if the user was removed from the IDP, which means they no longer are a valid user
		return nil, fmt.Errorf("%w: failed to update identity info from provider: %s", internal.ErrUnauthorized, err)
	}

	token, err := data.CreateIdentityToken(rCtx.DBTxn, rCtx.Authenticated.User.ID)
	if err != nil {
		return nil, err
	}

	return &api.CreateTokenResponse{Token: token.Token, Expires: api.Time(token.Expires)}, nil
}

// updateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) updateIdentityInfoFromProvider(rCtx RequestContext) error {
	provider, redirectURL, err := getContextProviderIdentity(rCtx)
	if err != nil {
		return err
	}

	if provider.Name == models.InternalInfraProviderName || provider.Kind == models.ProviderKindInfra {
		return nil
	}

	oidc, err := a.providerClient(rCtx.Request.Context(), provider, redirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	return updateIdentityInfoFromProvider(rCtx, oidc)
}

func getContextProviderIdentity(rCtx RequestContext) (*models.Provider, string, error) {
	providerUser, err := data.GetProviderUser(
		rCtx.DBTxn,
		rCtx.Authenticated.AccessKey.ProviderID,
		rCtx.Authenticated.User.ID)
	if err != nil {
		return nil, "", err
	}

	provider, err := data.GetProvider(rCtx.DBTxn, data.ByID(providerUser.ProviderID))
	if err != nil {
		return nil, "", fmt.Errorf("user info provider: %w", err)
	}

	return provider, providerUser.RedirectURL, nil
}

// updateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func updateIdentityInfoFromProvider(rCtx RequestContext, oidc providers.OIDCClient) error {
	ctx := rCtx.Request.Context()
	db := rCtx.DBTxn

	provider, err := data.GetProvider(db, data.ByID(rCtx.Authenticated.AccessKey.ProviderID))
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

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
