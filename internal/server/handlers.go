package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/infrahq/secrets"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

type API struct {
	t          *Telemetry
	server     *Server
	migrations []apiMigration
	openAPIDoc openapi3.T
}

func (a *API) CreateToken(c *gin.Context, r *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	rCtx := getRequestContext(c)

	if rCtx.Authenticated.User != nil {
		err := a.UpdateIdentityInfoFromProvider(c)
		if err != nil {
			// this will fail if the user was removed from the IDP, which means they no longer are a valid user
			return nil, fmt.Errorf("%w: failed to update identity info from provider: %s", internal.ErrUnauthorized, err)
		}

		token, err := access.CreateToken(rCtx)
		if err != nil {
			return nil, err
		}

		return &api.CreateTokenResponse{Token: token.Token, Expires: api.Time(token.Expires)}, nil
	}

	return nil, fmt.Errorf("%w: no identity found in access key", internal.ErrUnauthorized)
}

type WellKnownJWKResponse struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

func wellKnownJWKsHandler(c *gin.Context, _ *api.EmptyRequest) (WellKnownJWKResponse, error) {
	rCtx := getRequestContext(c)
	keys, err := access.GetPublicJWK(rCtx)
	if err != nil {
		return WellKnownJWKResponse{}, err
	}

	return WellKnownJWKResponse{Keys: keys}, nil
}

func (a *API) ListAccessKeys(c *gin.Context, r *api.ListAccessKeysRequest) (*api.ListResponse[api.AccessKey], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	accessKeys, err := access.ListAccessKeys(c, r.UserID, r.Name, r.ShowExpired, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(accessKeys, models.PaginationToResponse(p), func(accessKey models.AccessKey) api.AccessKey {
		return *accessKey.ToAPI()
	})

	return result, nil
}

func (a *API) DeleteAccessKey(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteAccessKey(c, r.ID)
}

func (a *API) CreateAccessKey(c *gin.Context, r *api.CreateAccessKeyRequest) (*api.CreateAccessKeyResponse, error) {
	accessKey := &models.AccessKey{
		IssuedFor:         r.UserID,
		Name:              r.Name,
		ProviderID:        access.InfraProvider(c).ID,
		ExpiresAt:         time.Now().UTC().Add(time.Duration(r.TTL)),
		Extension:         time.Duration(r.ExtensionDeadline),
		ExtensionDeadline: time.Now().UTC().Add(time.Duration(r.ExtensionDeadline)),
	}

	raw, err := access.CreateAccessKey(c, accessKey)
	if err != nil {
		return nil, err
	}

	return &api.CreateAccessKeyResponse{
		ID:                accessKey.ID,
		Created:           api.Time(accessKey.CreatedAt),
		Name:              accessKey.Name,
		IssuedFor:         accessKey.IssuedFor,
		Expires:           api.Time(accessKey.ExpiresAt),
		ExtensionDeadline: api.Time(accessKey.ExtensionDeadline),
		AccessKey:         raw,
	}, nil
}

func (a *API) Signup(c *gin.Context, r *api.SignupRequest) (*api.SignupResponse, error) {
	if !a.server.options.EnableSignup {
		return nil, fmt.Errorf("%w: signup is disabled", internal.ErrBadRequest)
	}

	org := &models.Organization{Name: r.Org}
	keyExpires := time.Now().UTC().Add(a.server.options.SessionDuration)

	identity, bearer, err := access.Signup(c, keyExpires, r.Name, r.Password, org)
	if err != nil {
		return nil, err
	}

	setAuthCookie(c, bearer, keyExpires)

	a.t.User(identity.ID.String(), r.Name)
	a.t.Alias(identity.ID.String())
	a.t.Event("signup", identity.ID.String(), Properties{})

	return &api.SignupResponse{
		User:         identity.ToAPI(),
		Organization: org.ToAPI(),
	}, nil
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	var loginMethod authn.LoginMethod

	expires := time.Now().UTC().Add(a.server.options.SessionDuration)

	switch {
	case r.AccessKey != "":
		loginMethod = authn.NewKeyExchangeAuthentication(r.AccessKey, expires)
	case r.PasswordCredentials != nil:
		loginMethod = authn.NewPasswordCredentialAuthentication(r.PasswordCredentials.Name, r.PasswordCredentials.Password)
	case r.OIDC != nil:
		provider, err := access.GetProvider(c, r.OIDC.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("invalid identity provider: %w", err)
		}

		providerClient, err := a.providerClient(c, provider, r.OIDC.RedirectURL)
		if err != nil {
			return nil, fmt.Errorf("update provider client: %w", err)
		}

		loginMethod = authn.NewOIDCAuthentication(r.OIDC.ProviderID, r.OIDC.RedirectURL, r.OIDC.Code, providerClient)
	default:
		// make sure to always fail by default
		return nil, fmt.Errorf("%w: missing login credentials", internal.ErrBadRequest)
	}

	// do the actual login now that we know the method selected
	key, bearer, requiresUpdate, err := access.Login(c, loginMethod, expires, a.server.options.SessionExtensionDeadline)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			// the user should be shown this explicitly
			// this means an external request failed, probably to an IDP
			return nil, err
		}
		// all other failures from login should result in an unauthorized response
		return nil, fmt.Errorf("%w: login failed: %v", internal.ErrUnauthorized, err)
	}

	setAuthCookie(c, bearer, expires)

	a.t.Event("login", key.IssuedFor.String(), Properties{"method": loginMethod.Name()})

	return &api.LoginResponse{UserID: key.IssuedFor, Name: key.IssuedForIdentity.Name, AccessKey: bearer, Expires: api.Time(expires), PasswordUpdateRequired: requiresUpdate}, nil
}

func (a *API) Logout(c *gin.Context, _ *api.EmptyRequest) (*api.EmptyResponse, error) {
	err := access.DeleteRequestAccessKey(getRequestContext(c))
	if err != nil {
		return nil, err
	}

	deleteAuthCookie(c)
	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.FullVersion()}, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) UpdateIdentityInfoFromProvider(c *gin.Context) error {
	rCtx := getRequestContext(c)
	provider, redirectURL, err := access.GetContextProviderIdentity(rCtx)
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

	return access.UpdateIdentityInfoFromProvider(rCtx, oidc)
}

func (a *API) providerClient(ctx context.Context, provider *models.Provider, redirectURL string) (providers.OIDCClient, error) {
	if c := providers.OIDCClientFromContext(ctx); c != nil {
		// oidc is added to the context during unit tests
		return c, nil
	}

	clientSecret, err := secrets.GetSecret(string(provider.ClientSecret), a.server.secrets)
	if err != nil {
		logging.Debugf("could not get client secret: %s", err)
		return nil, fmt.Errorf("client secret not found")
	}

	return providers.NewOIDCClient(*provider, clientSecret, redirectURL), nil
}
