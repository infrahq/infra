package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

type oidcAuthn struct {
	Provider            *models.Provider
	RedirectURL         string
	Code                string
	OIDCProviderClient  providers.OIDCClient
	AllowedLoginDomains []string
}

func NewOIDCAuthentication(provider *models.Provider, redirectURL string, code string, oidcProviderClient providers.OIDCClient, allowedLoginDomains []string) (LoginMethod, error) {
	if provider == nil {
		return nil, fmt.Errorf("nil provider in oidc authentication")
	}
	return &oidcAuthn{
		Provider:            provider,
		RedirectURL:         redirectURL,
		Code:                code,
		OIDCProviderClient:  oidcProviderClient,
		AllowedLoginDomains: allowedLoginDomains,
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

	if len(a.AllowedLoginDomains) > 0 {
		// get the domain of the email
		at := strings.LastIndex(idpAuth.Email, "@") // get the last @ since the email spec allows for multiple @s
		if at == -1 {
			return AuthenticatedIdentity{}, fmt.Errorf("%s is an invalid email address", idpAuth.Email)
		}
		domain := idpAuth.Email[at+1:]
		if !slices.Contains(a.AllowedLoginDomains, domain) {
			return AuthenticatedIdentity{}, fmt.Errorf("%s is not an allowed email domain", domain)
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
