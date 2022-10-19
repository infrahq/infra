package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

type oidcAuthn struct {
	ProviderID         uid.ID
	RedirectURL        string
	Code               string
	OIDCProviderClient providers.OIDCClient
}

func NewOIDCAuthentication(providerID uid.ID, redirectURL string, code string, oidcProviderClient providers.OIDCClient) LoginMethod {
	return &oidcAuthn{
		ProviderID:         providerID,
		RedirectURL:        redirectURL,
		Code:               code,
		OIDCProviderClient: oidcProviderClient,
	}
}

func (a *oidcAuthn) Authenticate(ctx context.Context, db data.GormTxn, requestedExpiry time.Time) (AuthenticatedIdentity, error) {
	provider, err := data.GetProvider(db, data.ByID(a.ProviderID))
	if err != nil {
		return AuthenticatedIdentity{}, err
	}

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	accessToken, refreshToken, expiry, email, err := a.OIDCProviderClient.ExchangeAuthCodeForProviderTokens(ctx, a.Code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return AuthenticatedIdentity{}, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return AuthenticatedIdentity{}, fmt.Errorf("exhange code for tokens: %w", err)
	}

	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: email, LoadGroups: true})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return AuthenticatedIdentity{}, fmt.Errorf("get user: %w", err)
		}

		identity = &models.Identity{Name: email}

		if err := data.CreateIdentity(db, identity); err != nil {
			return AuthenticatedIdentity{}, fmt.Errorf("create user: %w", err)
		}
	}

	providerUser, err := data.CreateProviderUser(db, provider, identity)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("add user for provider login: %w", err)
	}

	providerUser.RedirectURL = a.RedirectURL
	providerUser.AccessToken = models.EncryptedAtRest(accessToken)
	providerUser.RefreshToken = models.EncryptedAtRest(refreshToken)
	providerUser.ExpiresAt = expiry
	err = data.UpdateProviderUser(db, providerUser)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("UpdateProviderUser: %w", err)
	}

	// update users attributes (such as groups) from the IDP
	err = data.SyncProviderUser(ctx, db, identity, provider, a.OIDCProviderClient)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("sync user on login: %w", err)
	}

	return AuthenticatedIdentity{
		Identity:      identity,
		Provider:      provider,
		SessionExpiry: requestedExpiry,
	}, nil
}

func (a *oidcAuthn) Name() string {
	return "oidc"
}
