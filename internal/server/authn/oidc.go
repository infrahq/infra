package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/exp/slices"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

type oidcAuthn struct {
	Provider           *models.Provider
	RedirectURL        string
	Code               string
	OIDCProviderClient providers.OIDCClient
	AllowedDomains     []string
}

func NewOIDCAuthentication(provider *models.Provider, redirectURL string, code string, oidcProviderClient providers.OIDCClient, allowedDomains []string) (LoginMethod, error) {
	if provider == nil {
		return nil, fmt.Errorf("nil provider in oidc authentication")
	}
	return &oidcAuthn{
		Provider:           provider,
		RedirectURL:        redirectURL,
		Code:               code,
		OIDCProviderClient: oidcProviderClient,
		AllowedDomains:     allowedDomains,
	}, nil
}

func (a *oidcAuthn) Authenticate(ctx context.Context, db *data.Transaction, requestedExpiry time.Time) (AuthenticatedIdentity, error) {
	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	idpAuth, err := a.OIDCProviderClient.ExchangeAuthCodeForProviderTokens(ctx, a.Code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return AuthenticatedIdentity{}, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return AuthenticatedIdentity{}, fmt.Errorf("exhange code for tokens: %w", err)
	}

	if a.Provider.Managed {
		// this is a social login, check if they can access this org
		domain, err := email.Domain(idpAuth.Email)
		if err != nil {
			return AuthenticatedIdentity{}, err
		}
		if !slices.Contains(a.AllowedDomains, domain) {
			// check if the user has been added manually
			_, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: idpAuth.Email})
			if err != nil {
				if errors.Is(err, internal.ErrNotFound) {
					return AuthenticatedIdentity{}, fmt.Errorf("%s is not an allowed email domain or existing user", domain)
				}
				// someting else went wrong getting the user
				return AuthenticatedIdentity{}, fmt.Errorf("check user identity on social oidc login: %w", err)
			}
		}
	}

	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: idpAuth.Email, LoadGroups: true})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return AuthenticatedIdentity{}, fmt.Errorf("get user: %w", err)
		}

		identity = &models.Identity{Name: idpAuth.Email}

		if err := data.CreateIdentity(db, identity); err != nil {
			return AuthenticatedIdentity{}, fmt.Errorf("create user: %w", err)
		}
	}

	providerUser, err := data.CreateProviderUser(db, a.Provider, identity)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("add user for provider login: %w", err)
	}

	providerUser.RedirectURL = a.RedirectURL
	providerUser.AccessToken = models.EncryptedAtRest(idpAuth.AccessToken)
	providerUser.RefreshToken = models.EncryptedAtRest(idpAuth.RefreshToken)
	providerUser.ExpiresAt = idpAuth.AccessTokenExpiry
	err = data.UpdateProviderUser(db, providerUser)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("UpdateProviderUser: %w", err)
	}

	// update users attributes (such as groups) from the IDP
	err = data.SyncProviderUser(ctx, db, identity, a.Provider, a.OIDCProviderClient)
	if err != nil {
		return AuthenticatedIdentity{}, fmt.Errorf("sync user on login: %w", err)
	}

	return AuthenticatedIdentity{
		Identity:      identity,
		Provider:      a.Provider,
		SessionExpiry: requestedExpiry,
	}, nil
}

func (a *oidcAuthn) Name() string {
	return "oidc"
}
