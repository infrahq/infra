package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: remove method receiver
func (a *API) CreateToken(c *gin.Context, _ *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	if access.AuthenticatedIdentity(c) != nil {
		err := a.updateIdentityInfoFromProvider(c)
		if err != nil {
			// this will fail if the user was removed from the IDP, which means they no longer are a valid user
			return nil, fmt.Errorf("%w: failed to update identity info from provider: %s", internal.ErrUnauthorized, err)
		}

		token, err := access.CreateToken(c)
		if err != nil {
			return nil, err
		}

		return &api.CreateTokenResponse{Token: token.Token, Expires: api.Time(token.Expires)}, nil
	}

	return nil, fmt.Errorf("%w: no identity found in access key", internal.ErrUnauthorized)
}

// updateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) updateIdentityInfoFromProvider(c *gin.Context) error {
	provider, redirectURL, err := access.GetContextProviderIdentity(c)
	if err != nil {
		return err
	}

	if provider.Name == models.InternalInfraProviderName || provider.Kind == models.ProviderKindInfra {
		return nil
	}

	oidc, err := a.providerClient(c, provider, redirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	return access.UpdateIdentityInfoFromProvider(c, oidc)
}
