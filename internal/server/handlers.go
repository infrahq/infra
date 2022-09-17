package server

import (
	"context"
	"encoding/base64"
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
	"github.com/infrahq/infra/internal/server/email"
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
		err := a.UpdateIdentityInfoFromProvider(rCtx)
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

var wellKnownJWKsRoute = route[api.EmptyRequest, WellKnownJWKResponse]{
	handler:                    wellKnownJWKsHandler,
	omitFromDocs:               true,
	omitFromTelemetry:          true,
	infraVersionHeaderOptional: true,
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

func (a *API) ListAccessKeys(c *gin.Context, r *api.ListAccessKeysRequest) (*api.ListResponse[api.AccessKey], error) {
	p := PaginationFromRequest(r.PaginationRequest)
	accessKeys, err := access.ListAccessKeys(c, r.UserID, r.Name, r.ShowExpired, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(accessKeys, PaginationToResponse(p), func(accessKey models.AccessKey) api.AccessKey {
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

	keyExpires := time.Now().UTC().Add(a.server.options.SessionDuration)

	suDetails := access.SignupDetails{
		Name:      r.Name,
		Password:  r.Password,
		Org:       &models.Organization{Name: r.Org.Name},
		SubDomain: r.Org.Subdomain,
	}
	identity, bearer, err := access.Signup(c, keyExpires, a.server.options.BaseDomain, suDetails)
	if err != nil {
		return nil, err
	}

	/*
		This cookie is set to send on all infra domains, make it expire quickly to prevent an unexpected org being set on requests to other orgs.
		This signup cookie sets the authentication for the next call made to the org and will be exchanged for a long-term auth cookie.
		We have to set this short lived sign-up auth cookie to give the user a valid session on sign-up.
		Since the signup is on the base domain we have to set this cookie there,
		but we want auth cookies to only be sent to their respective orgs so they must be set on their org specific sub-domain after redirect.
	*/
	cookie := cookieConfig{
		Name:    cookieSignupName,
		Value:   bearer,
		Domain:  a.server.options.BaseDomain,
		Expires: time.Now().Add(1 * time.Minute),
	}
	setCookie(c, cookie)

	a.t.User(identity.ID.String(), r.Name)
	a.t.Org(suDetails.Org.ID.String(), identity.ID.String(), suDetails.Org.Name)
	a.t.Event("signup", identity.ID.String(), Properties{})

	link := fmt.Sprintf("https://%s", suDetails.Org.Domain)
	err = email.SendSignupEmail("", r.Name, email.SignupData{
		Link:        link,
		WrappedLink: wrapLinkWithVerification(link, suDetails.Org.Domain, identity.VerificationToken),
	})
	if err != nil {
		// if email failed, continue on anyway.
		logging.L.Error().Err(err).Msg("could not send signup email")
	}

	return &api.SignupResponse{
		User:         identity.ToAPI(),
		Organization: suDetails.Org.ToAPI(),
	}, nil
}

func wrapLinkWithVerification(link, domain, verificationToken string) string {
	link = base64.URLEncoding.EncodeToString([]byte(link))
	return fmt.Sprintf("https://%s/link?vt=%s&r=%s", domain, verificationToken, link)
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	rCtx := getRequestContext(c)

	var loginMethod authn.LoginMethod
	switch {
	case r.AccessKey != "":
		loginMethod = authn.NewKeyExchangeAuthentication(r.AccessKey)
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
	expires := time.Now().UTC().Add(a.server.options.SessionDuration)
	result, err := authn.Login(rCtx.Request.Context(), rCtx.DBTxn, loginMethod, expires, a.server.options.SessionExtensionDeadline)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			// the user should be shown this explicitly
			// this means an external request failed, probably to an IDP
			return nil, err
		}
		// all other failures from login should result in an unauthorized response
		return nil, fmt.Errorf("%w: login failed: %v", internal.ErrUnauthorized, err)
	}

	cookie := cookieConfig{
		Name:    cookieAuthorizationName,
		Value:   result.Bearer,
		Domain:  c.Request.Host,
		Expires: result.AccessKey.ExpiresAt,
	}
	setCookie(c, cookie)

	key := result.AccessKey
	a.t.User(key.IssuedFor.String(), result.User.Name)
	a.t.OrgMembership(key.OrganizationID.String(), key.IssuedFor.String())
	a.t.Event("login", key.IssuedFor.String(), Properties{"method": loginMethod.Name()})

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

	deleteCookie(c, cookieAuthorizationName, c.Request.Host)
	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.FullVersion()}, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) UpdateIdentityInfoFromProvider(rCtx access.RequestContext) error {
	provider, redirectURL, err := access.GetContextProviderIdentity(rCtx)
	if err != nil {
		return err
	}

	if provider.Kind == models.ProviderKindInfra {
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
