package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

type API struct {
	t          *Telemetry
	server     *Server
	migrations []apiMigration
	openAPIDoc openapi3.T
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

func (a *API) SignupEnabled(c *gin.Context, _ *api.EmptyRequest) (*api.SignupEnabledResponse, error) {
	if !a.server.options.EnableSignup {
		return &api.SignupEnabledResponse{Enabled: false}, nil
	}

	signupEnabled, err := access.SignupEnabled(c)
	if err != nil {
		return nil, err
	}

	return &api.SignupEnabledResponse{Enabled: signupEnabled}, nil
}

func (a *API) Signup(c *gin.Context, r *api.SignupRequest) (*api.User, error) {
	if !a.server.options.EnableSignup {
		return nil, fmt.Errorf("%w: signup is disabled", internal.ErrBadRequest)
	}

	signupEnabled, err := access.SignupEnabled(c)
	if err != nil {
		return nil, err
	}

	if !signupEnabled {
		return nil, fmt.Errorf("%w: signup is disabled", internal.ErrBadRequest)
	}

	identity, err := access.Signup(c, r.Name, r.Password)
	if err != nil {
		return nil, err
	}

	a.t.User(identity.ID.String(), r.Name)
	a.t.Alias(identity.ID.String())
	a.t.Event("signup", identity.ID.String(), Properties{})

	return identity.ToAPI(), nil
}

// TODO: remove method receiver
func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	rCtx := getRequestContext(c)
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

		providerClient, err := newProviderOIDCClient(rCtx, provider, r.OIDC.RedirectURL)
		if err != nil {
			return nil, fmt.Errorf("update provider client: %w", err)
		}

		loginMethod = authn.NewOIDCAuthentication(r.OIDC.ProviderID, r.OIDC.RedirectURL, r.OIDC.Code, providerClient)
	default:
		// make sure to always fail by default
		return nil, fmt.Errorf("%w: missing login credentials", internal.ErrBadRequest)
	}

	key, bearer, err := authn.Login(
		rCtx.Request.Context(),
		rCtx.DBTxn,
		loginMethod,
		expires,
		a.server.options.SessionExtensionDeadline)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			// the user should be shown this explicitly
			// this means an external request failed, probably to an IDP
			return nil, err
		}
		// all other failures from login should result in an unauthorized response
		return nil, fmt.Errorf("%w: login failed: %v", internal.ErrUnauthorized, err)
	}

	// In the case of username/password credentials,
	// the login may fail if the password presented was a one-time password that has been used.
	// This can be removed when #1441 is resolved
	requiresUpdate, err := loginMethod.RequiresUpdate(rCtx.DBTxn)
	if err != nil {
		return nil, err
	}

	setAuthCookie(c, bearer, expires)

	a.t.Event("login", key.IssuedFor.String(), Properties{"method": loginMethod.Name()})

	return &api.LoginResponse{
		UserID:                 key.IssuedFor,
		Name:                   key.IssuedForIdentity.Name,
		AccessKey:              bearer,
		Expires:                api.Time(expires),
		PasswordUpdateRequired: requiresUpdate,
	}, nil
}

func Logout(c *gin.Context, r *api.EmptyRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)

	if err := data.DeleteAccessKey(rCtx.DBTxn, rCtx.Authenticated.AccessKey.ID); err != nil {
		return nil, err
	}

	deleteAuthCookie(c.Writer)
	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.FullVersion()}, nil
}

func newProviderOIDCClient(rCtx RequestContext, provider *models.Provider, redirectURL string) (providers.OIDCClient, error) {
	if c := providers.OIDCClientFromContext(rCtx.Request.Context()); c != nil {
		// oidc is added to the context during unit tests
		return c, nil
	}

	clientSecret, err := rCtx.GetSecret(string(provider.ClientSecret))
	if err != nil {
		logging.Debugf("could not get client secret: %s", err)
		return nil, fmt.Errorf("client secret not found")
	}

	return providers.NewOIDCClient(*provider, clientSecret, redirectURL), nil
}
