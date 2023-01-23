package server

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/openapi3"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/redis"
)

type API struct {
	t          *Telemetry
	server     *Server
	openAPIDoc openapi3.Doc
	versions   map[routeIdentifier][]routeVersion
}

type routeVersion struct {
	version *semver.Version
	handler func(c *gin.Context)
}

func addVersionHandler[Req, Res any](a *API, method, path, version string, routeDef route[Req, Res]) {
	if a.versions == nil {
		a.versions = make(map[routeIdentifier][]routeVersion)
	}

	routeDef.routeSettings.omitFromDocs = true

	key := routeIdentifier{method: method, path: path}
	a.versions[key] = append(a.versions[key], routeVersion{
		version: semver.MustParse(version),
		handler: func(c *gin.Context) {
			wrapped := wrapRoute(a, key, routeDef)
			if err := wrapped(c); err != nil {
				sendAPIError(c.Writer, c.Request, err)
			}
		},
	})
}

var createTokenRoute = route[api.EmptyRequest, *api.CreateTokenResponse]{
	routeSettings: routeSettings{idpSync: true},
	handler:       CreateToken,
}

func CreateToken(c *gin.Context, r *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	rCtx := getRequestContext(c)

	if rCtx.Authenticated.User == nil {
		return nil, fmt.Errorf("no authenticated user")
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
	keys, err := getPublicJWK(rCtx)
	if err != nil {
		return WellKnownJWKResponse{}, err
	}

	return WellKnownJWKResponse{Keys: keys}, nil
}

func getPublicJWK(rCtx access.RequestContext) ([]jose.JSONWebKey, error) {
	settings, err := data.GetSettings(rCtx.DBTxn)
	if err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	var pubKey jose.JSONWebKey
	if err := pubKey.UnmarshalJSON(settings.PublicJWK); err != nil {
		return nil, fmt.Errorf("could not get JWKs: %w", err)
	}

	return []jose.JSONWebKey{pubKey}, nil
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

		providerClient, err := a.server.providerClient(rCtx.Request.Context(), provider, r.OIDC.RedirectURL)
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
		Domain:  rCtx.Request.Host,
		Expires: result.AccessKey.ExpiresAt,
	}
	setCookie(rCtx.Request, rCtx.Response.HTTPWriter, cookie)

	key := result.AccessKey
	a.t.User(key.IssuedFor.String(), result.User.Name)
	a.t.OrgMembership(key.OrganizationID.String(), key.IssuedFor.String())
	a.t.Event("login", key.IssuedFor.String(), key.OrganizationID.String(), Properties{
		"method": loginMethod.Name(),
		"email":  result.User.Name,
	})

	// Update the request context so that logging middleware can include the userID
	rCtx.Authenticated.User = result.User
	rCtx.Response.LoginUserID = result.User.ID

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
	// does not need authorization check, this action is limited to the calling key
	rCtx := getRequestContext(c)
	id := rCtx.Authenticated.AccessKey.ID
	err := data.DeleteAccessKeys(rCtx.DBTxn, data.DeleteAccessKeysOptions{ByID: id})
	if err != nil {
		return nil, err
	}

	deleteCookie(c.Request, rCtx.Response.HTTPWriter, cookieAuthorizationName, c.Request.Host)
	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.FullVersion()}, nil
}
