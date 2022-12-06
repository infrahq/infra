package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/internal/server/redis"
)

type API struct {
	t          *Telemetry
	server     *Server
	migrations []apiMigration
	openAPIDoc openapi3.T
}

func (a *API) CreateToken(c *gin.Context, r *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	rCtx := getRequestContext(c)

	if rCtx.Authenticated.User == nil {
		return nil, fmt.Errorf("no authenticated user")
	}
	err := a.UpdateIdentityInfoFromProvider(rCtx)
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

var wellKnownJWKsRoute = route[api.EmptyRequest, WellKnownJWKResponse]{
	handler: wellKnownJWKsHandler,
	routeSettings: routeSettings{
		omitFromDocs:               true,
		omitFromTelemetry:          true,
		infraVersionHeaderOptional: true,
		txnOptions:                 &sql.TxOptions{ReadOnly: true},
	},
}

func wellKnownJWKsHandler(c *gin.Context, _ *api.EmptyRequest) (WellKnownJWKResponse, error) {
	rCtx := getRequestContext(c)
	keys, err := access.GetPublicJWK(rCtx)
	if err != nil {
		return WellKnownJWKResponse{}, err
	}

	return WellKnownJWKResponse{Keys: keys}, nil
}

type WellKnownJWKResponse struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

func wrapLinkWithVerification(link, domain, verificationToken string) string {
	link = base64.URLEncoding.EncodeToString([]byte(link))
	return fmt.Sprintf("https://%s/link?vt=%s&r=%s", domain, verificationToken, link)
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	rCtx := getRequestContext(c)

	var onSuccess, onFailure func()

	var loginMethod authn.LoginMethod
	switch {
	case r.AccessKey != "":
		loginMethod = authn.NewKeyExchangeAuthentication(r.AccessKey)
	case r.PasswordCredentials != nil:
		if err := redis.NewLimiter(a.server.redis).RateOK(r.PasswordCredentials.Name, 10); err != nil {
			return nil, err
		}

		usernameWithOrganization := fmt.Sprintf("%s:%s", r.PasswordCredentials.Name, rCtx.Authenticated.Organization.ID)
		limiter := redis.NewLimiter(a.server.redis)
		if err := limiter.LoginOK(usernameWithOrganization); err != nil {
			return nil, err
		}

		onSuccess = func() {
			limiter.LoginGood(usernameWithOrganization)
		}

		onFailure = func() {
			limiter.LoginBad(usernameWithOrganization, 10)
		}

		loginMethod = authn.NewPasswordCredentialAuthentication(r.PasswordCredentials.Name, r.PasswordCredentials.Password)
	case r.OIDC != nil:
		var provider *models.Provider
		if r.OIDC.ProviderID == models.InternalGoogleProviderID {
			if a.server.Google == nil {
				return nil, fmt.Errorf("%w: google login is not configured, provider id must be specified for oidc login", internal.ErrBadRequest)
			}
			// default to Google social login
			provider = a.server.Google
		} else {
			var err error
			provider, err = data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{ByID: r.OIDC.ProviderID})
			if err != nil {
				return nil, fmt.Errorf("invalid identity provider: %w", err)
			}
		}

		providerClient, err := a.providerClient(c, provider, r.OIDC.RedirectURL)
		if err != nil {
			return nil, fmt.Errorf("login provider client: %w", err)
		}

		loginMethod, err = authn.NewOIDCAuthentication(
			provider,
			r.OIDC.RedirectURL,
			r.OIDC.Code,
			providerClient,
			rCtx.Authenticated.Organization.AllowedDomains,
		)
		if err != nil {
			return nil, err
		}
	default:
		// make sure to always fail by default
		return nil, fmt.Errorf("%w: missing login credentials", internal.ErrBadRequest)
	}

	// do the actual login now that we know the method selected
	expires := time.Now().UTC().Add(a.server.options.SessionDuration)
	result, err := authn.Login(rCtx.Request.Context(), rCtx.DBTxn, loginMethod, expires, a.server.options.SessionInactivityTimeout)
	if err != nil {
		if onFailure != nil {
			onFailure()
		}

		if errors.Is(err, internal.ErrBadGateway) {
			// the user should be shown this explicitly
			// this means an external request failed, probably to an IDP
			return nil, err
		}
		// all other failures from login should result in an unauthorized response
		return nil, fmt.Errorf("%w: login failed: %v", internal.ErrUnauthorized, err)
	}

	if onSuccess != nil {
		onSuccess()
	}

	cookie := cookieConfig{
		Name:    cookieAuthorizationName,
		Value:   result.Bearer,
		Domain:  c.Request.Host,
		Expires: result.AccessKey.ExpiresAt,
	}
	setCookie(c.Request, c.Writer, cookie)

	key := result.AccessKey
	a.t.User(key.IssuedFor.String(), result.User.Name)
	a.t.OrgMembership(key.OrganizationID.String(), key.IssuedFor.String())
	a.t.Event("login", key.IssuedFor.String(), key.OrganizationID.String(), Properties{"method": loginMethod.Name()})

	// Update the request context so that logging middleware can include the userID
	rCtx.Authenticated.User = result.User
	c.Set(access.RequestContextKey, rCtx)

	return &api.LoginResponse{
		UserID:                 key.IssuedFor,
		Name:                   key.IssuedForName,
		AccessKey:              result.Bearer,
		Expires:                api.Time(key.ExpiresAt),
		PasswordUpdateRequired: result.CredentialUpdateRequired,
		OrganizationName:       result.OrganizationName,
	}, nil
}

func (a *API) Logout(c *gin.Context, _ *api.EmptyRequest) (*api.EmptyResponse, error) {
	err := access.DeleteRequestAccessKey(getRequestContext(c))
	if err != nil {
		return nil, err
	}

	deleteCookie(c.Writer, cookieAuthorizationName, c.Request.Host)
	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.FullVersion()}, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) UpdateIdentityInfoFromProvider(rCtx access.RequestContext) error {
	// does not need access check, this action is limited to the calling user
	identity := rCtx.Authenticated.User
	if identity == nil {
		return errors.New("user does not have session with an identity provider")
	}

	var provider *models.Provider
	if a.server.Google != nil && rCtx.Authenticated.AccessKey.ProviderID == a.server.Google.ID {
		provider = a.server.Google
	} else {
		var err error
		provider, err = data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{
			ByID: rCtx.Authenticated.AccessKey.ProviderID,
		})
		if err != nil {
			return fmt.Errorf("failed to get provider for user info: %w", err)
		}

		if provider.Kind == models.ProviderKindInfra {
			// no external verification needed
			logging.L.Trace().Msg("skipped verifying identity within infra provider, not required")
			return nil
		}
	}

	providerUser, err := data.GetProviderUser(rCtx.DBTxn, rCtx.Authenticated.AccessKey.ProviderID, identity.ID)
	if err != nil {
		return fmt.Errorf("failed to get provider user to update: %w", err)
	}

	oidc, err := a.providerClient(rCtx.Request.Context(), provider, providerUser.RedirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	return access.UpdateIdentityInfoFromProvider(rCtx, provider, oidc)
}

func (a *API) providerClient(ctx context.Context, provider *models.Provider, redirectURL string) (providers.OIDCClient, error) {
	if c := providers.OIDCClientFromContext(ctx); c != nil {
		// oidc is added to the context during unit tests
		return c, nil
	}

	return providers.NewOIDCClient(*provider, string(provider.ClientSecret), redirectURL), nil
}
