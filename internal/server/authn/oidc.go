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
	"github.com/infrahq/infra/uid"
)

type OIDCAuthn struct {
	ProviderID         uid.ID
	RedirectURL        string
	Code               string
	OIDCProviderClient providers.OIDCClient
}

func NewOIDCAuthentication(providerID uid.ID, redirectURL string, code string, oidcProviderClient providers.OIDCClient) LoginMethod {
	return &OIDCAuthn{
		ProviderID:         providerID,
		RedirectURL:        redirectURL,
		Code:               code,
		OIDCProviderClient: oidcProviderClient,
	}
}

func (a *OIDCAuthn) Authenticate(ctx context.Context, db *data.Transaction, requestedExpiry time.Time) (AuthenticatedIdentity, error) {
	provider, err := data.GetProvider(db, data.GetProviderOptions{ByID: a.ProviderID})
	if err != nil {
		return AuthenticatedIdentity{}, err
	}

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	idpAuth, err := a.OIDCProviderClient.ExchangeAuthCodeForProviderTokens(ctx, a.Code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return AuthenticatedIdentity{}, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return AuthenticatedIdentity{}, fmt.Errorf("exhange code for tokens: %w", err)
	}

	if provider.Managed {
		// this is a social login
		domain, err := email.GetDomain(idpAuth.Email)
		if err != nil {
			return AuthenticatedIdentity{}, err
		}
		if !slices.Contains(provider.AllowedDomains, domain) {
			// check if the user has been added manually
			_, err := data.GetIdentity(db, data.GetIdentityOptions{ByName: idpAuth.Email})
			if errors.Is(err, internal.ErrNotFound) {
				return AuthenticatedIdentity{}, fmt.Errorf("%s is not an allowed email domain or existing user", domain)
			}
			if err != nil {
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

	providerUser, err := data.CreateProviderUser(db, provider, identity)
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

func (a *OIDCAuthn) Name() string {
	return "oidc"
}
