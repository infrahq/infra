package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/secrets"
)

type API struct {
	t      *Telemetry
	server *Server
}

func (a *API) ListIdentities(c *gin.Context, r *api.ListIdentitiesRequest) ([]api.Identity, error) {
	identities, err := access.ListIdentities(c, r.Name)
	if err != nil {
		return nil, err
	}

	results := make([]api.Identity, len(identities))
	for i, identity := range identities {
		results[i] = *identity.ToAPI()
	}

	a.t.Event(c, "users.list")

	return results, nil
}

func (a *API) GetIdentity(c *gin.Context, r *api.Resource) (*api.Identity, error) {
	identity, err := access.GetIdentity(c, r.ID)
	if err != nil {
		return nil, err
	}

	return identity.ToAPI(), nil
}

func (a *API) CreateIdentity(c *gin.Context, r *api.CreateIdentityRequest) (*api.CreateIdentityResponse, error) {
	kind, err := models.ParseIdentityKind(r.Kind)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", internal.ErrBadRequest, err)
	}

	identity := &models.Identity{
		Name: r.Name,
		Kind: kind,
	}

	if err := access.CreateIdentity(c, identity); err != nil {
		return nil, err
	}

	_, err = access.CreateProviderUser(c, access.InfraProvider(c), identity)
	if err != nil {
		return nil, err
	}

	defaultGrant := &models.Grant{Subject: identity.PolyID(), Privilege: models.InfraUserRole, Resource: access.ResourceInfraAPI}
	if err := access.CreateGrant(c, defaultGrant); err != nil {
		return nil, err
	}

	resp := &api.CreateIdentityResponse{
		ID:         identity.ID,
		Name:       identity.Name,
		ProviderID: access.InfraProvider(c).ID,
	}

	if identity.Kind == models.UserKind {
		oneTimePassword, err := access.CreateCredential(c, *identity)
		if err != nil {
			return nil, err
		}

		resp.OneTimePassword = oneTimePassword
	}

	return resp, nil
}

func (a *API) UpdateIdentity(c *gin.Context, r *api.UpdateIdentityRequest) (*api.Identity, error) {
	// right now this endpoint can only update a user's credentials, so get the user identity
	identity, err := access.GetIdentity(c, r.ID)
	if err != nil {
		return nil, err
	}

	if identity.Kind != models.UserKind {
		return nil, fmt.Errorf("%w: machine identity has no password to update", internal.ErrBadRequest)
	}

	err = access.UpdateCredential(c, identity, r.Password)
	if err != nil {
		return nil, err
	}

	return identity.ToAPI(), nil
}

func (a *API) DeleteIdentity(c *gin.Context, r *api.Resource) error {
	return access.DeleteIdentity(c, r.ID)
}

func (a *API) ListIdentityGrants(c *gin.Context, r *api.Resource) ([]api.Grant, error) {
	grants, err := access.ListIdentityGrants(c, r.ID)
	if err != nil {
		return nil, err
	}

	results := make([]api.Grant, len(grants))
	for i, g := range grants {
		results[i] = *g.ToAPI()
	}

	a.t.Event(c, "user.grants.list", Properties{"id": r.ID.String()})

	return results, nil
}

func (a *API) ListIdentityGroups(c *gin.Context, r *api.Resource) ([]api.Group, error) {
	groups, err := access.ListIdentityGroups(c, r.ID)
	if err != nil {
		return nil, err
	}

	results := make([]api.Group, len(groups))
	for i, g := range groups {
		results[i] = *g.ToAPI()
	}

	a.t.Event(c, "user.groups.list", Properties{"id": r.ID.String()})

	return results, nil
}

func (a *API) ListGroups(c *gin.Context, r *api.ListGroupsRequest) ([]api.Group, error) {
	groups, err := access.ListGroups(c, r.Name)
	if err != nil {
		return nil, err
	}

	results := make([]api.Group, len(groups))
	for i, g := range groups {
		results[i] = *g.ToAPI()
	}

	a.t.Event(c, "groups.list")

	return results, nil
}

func (a *API) GetGroup(c *gin.Context, r *api.Resource) (*api.Group, error) {
	group, err := access.GetGroup(c, r.ID)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "group.get", Properties{"id": r.ID.String()})

	return group.ToAPI(), nil
}

func (a *API) CreateGroup(c *gin.Context, r *api.CreateGroupRequest) (*api.Group, error) {
	group := &models.Group{
		Name: r.Name,
	}

	err := access.CreateGroup(c, group)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "group.create", Properties{"id": group.ID.String()})

	return group.ToAPI(), nil
}

func (a *API) ListGroupGrants(c *gin.Context, r *api.Resource) ([]api.Grant, error) {
	grants, err := access.ListGroupGrants(c, r.ID)
	if err != nil {
		return nil, err
	}

	results := make([]api.Grant, len(grants))
	for i, d := range grants {
		results[i] = *d.ToAPI()
	}

	a.t.Event(c, "group.grants.list", Properties{"id": r.ID.String()})

	return results, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context, r *api.ListProvidersRequest) ([]api.Provider, error) {
	providers, err := access.ListProviders(c, r.Name)
	if err != nil {
		return nil, err
	}

	results := make([]api.Provider, len(providers))
	for i, p := range providers {
		results[i] = *p.ToAPI()
	}

	a.t.Event(c, "providers.list")

	return results, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context, r *api.Resource) (*api.Provider, error) {
	provider, err := access.GetProvider(c, r.ID)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "provider.get")

	return provider.ToAPI(), nil
}

var (
	dashAdminRemover = regexp.MustCompile(`(.*)\-admin(\.okta\.com)`)
	protocolRemover  = regexp.MustCompile(`http[s]?://`)
)

func cleanupURL(url string) string {
	url = strings.TrimSpace(url)
	url = dashAdminRemover.ReplaceAllString(url, "$1$2")
	url = protocolRemover.ReplaceAllString(url, "")

	return url
}

func (a *API) CreateProvider(c *gin.Context, r *api.CreateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Name:         r.Name,
		URL:          cleanupURL(r.URL),
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	err := access.CreateProvider(c, provider)
	if err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

func (a *API) UpdateProvider(c *gin.Context, r *api.UpdateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Model: models.Model{
			ID: r.ID,
		},
		Name:         r.Name,
		URL:          cleanupURL(r.URL),
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	err := access.SaveProvider(c, provider)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "provider.update", Properties{"id": provider.ID.String()})

	return provider.ToAPI(), nil
}

func (a *API) DeleteProvider(c *gin.Context, r *api.Resource) error {
	if err := access.DeleteProvider(c, r.ID); err != nil {
		return err
	}

	a.t.Event(c, "provider.delete", Properties{"id": r.ID.String()})

	return nil
}

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) ([]api.Destination, error) {
	destinations, err := access.ListDestinations(c, r.UniqueID, r.Name)
	if err != nil {
		return nil, err
	}

	results := make([]api.Destination, len(destinations))
	for i, d := range destinations {
		results[i] = *d.ToAPI()
	}

	a.t.Event(c, "destinations.list")

	return results, nil
}

// Introspect is used by clients to get info about the token they are using
func (a *API) Introspect(c *gin.Context, r *api.EmptyRequest) (*api.Introspect, error) {
	identity := access.CurrentIdentity(c)
	if identity != nil {
		return &api.Introspect{ID: identity.ID, Name: identity.Name, IdentityType: identity.Kind.String()}, nil
	}

	a.t.Event(c, "introspect")

	return nil, fmt.Errorf("no identity context found for token")
}

func (a *API) GetDestination(c *gin.Context, r *api.Resource) (*api.Destination, error) {
	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "destination.get", Properties{"id": r.ID.String()})

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(c *gin.Context, r *api.CreateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  r.Connection.CA,
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	a.t.Event(c, "destination.create", Properties{"id": destination.ID.String()})

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{
		Model: models.Model{
			ID: r.ID,
		},
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  r.Connection.CA,
	}

	if err := access.SaveDestination(c, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	a.t.Event(c, "destination.update", Properties{"id": destination.ID.String()})

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) error {
	if err := access.DeleteDestination(c, r.ID); err != nil {
		return err
	}

	a.t.Event(c, "destination.delete", Properties{"id": r.ID.String()})

	return nil
}

func (a *API) CreateToken(c *gin.Context, r *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	if access.CurrentIdentity(c) != nil {
		err := a.UpdateIdentityInfoFromProvider(c)
		if err != nil {
			return nil, fmt.Errorf("update ident info from provider: %w", err)
		}

		token, err := access.CreateToken(c)
		if err != nil {
			return nil, err
		}

		return &api.CreateTokenResponse{Token: token.Token, Expires: api.Time(token.Expires)}, nil
	}

	return nil, fmt.Errorf("no identity found in access key: %w", internal.ErrUnauthorized)
}

func (a *API) ListAccessKeys(c *gin.Context, r *api.ListAccessKeysRequest) ([]api.AccessKey, error) {
	accessKeys, err := access.ListAccessKeys(c, r.IdentityID, r.Name)
	if err != nil {
		return nil, err
	}

	results := make([]api.AccessKey, len(accessKeys))

	for i, a := range accessKeys {
		results[i] = api.AccessKey{
			ID:                a.ID,
			Name:              a.Name,
			Created:           api.Time(a.CreatedAt),
			IssuedFor:         a.IssuedFor,
			Expires:           api.Time(a.ExpiresAt),
			ExtensionDeadline: api.Time(a.ExtensionDeadline),
		}
	}

	a.t.Event(c, "accesskeys.list")

	return results, nil
}

func (a *API) DeleteAccessKey(c *gin.Context, r *api.Resource) error {
	if err := access.DeleteAccessKey(c, r.ID); err != nil {
		return err
	}

	a.t.Event(c, "accesskey.delete")

	return nil
}

func (a *API) CreateAccessKey(c *gin.Context, r *api.CreateAccessKeyRequest) (*api.CreateAccessKeyResponse, error) {
	accessKey := &models.AccessKey{
		IssuedFor:         r.IdentityID,
		Name:              r.Name,
		ExpiresAt:         time.Now().Add(time.Duration(r.TTL)).UTC(),
		Extension:         time.Duration(r.ExtensionDeadline),
		ExtensionDeadline: time.Now().Add(time.Duration(r.ExtensionDeadline)).UTC(),
	}

	raw, err := access.CreateAccessKey(c, accessKey, r.IdentityID)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "accesskey.create")

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

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) ([]api.Grant, error) {
	grants, err := access.ListGrants(c, r.Subject, r.Resource, r.Privilege)
	if err != nil {
		return nil, err
	}

	results := make([]api.Grant, len(grants))
	for i, r := range grants {
		results[i] = *r.ToAPI()
	}

	a.t.Event(c, "grants.list")

	return results, nil
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.CreateGrantRequest) (*api.Grant, error) {
	grant := &models.Grant{
		Resource:  r.Resource,
		Privilege: r.Privilege,
		Subject:   r.Subject,
	}

	err := access.CreateGrant(c, grant)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) DeleteGrant(c *gin.Context, r *api.Resource) error {
	if err := access.DeleteGrant(c, r.ID); err != nil {
		return err
	}

	a.t.Event(c, "grant.delete")

	return nil
}

func (a *API) SetupRequired(c *gin.Context, _ *api.EmptyRequest) (*api.SetupRequiredResponse, error) {
	setupRequired, err := access.SetupRequired(c)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "setuprequired")

	return &api.SetupRequiredResponse{
		Required: setupRequired,
	}, nil
}

func (a *API) Setup(c *gin.Context, _ *api.EmptyRequest) (*api.CreateAccessKeyResponse, error) {
	raw, accessKey, err := access.Setup(c)
	if err != nil {
		return nil, err
	}

	a.t.Event(c, "setup")

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

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	expires := time.Now().Add(a.server.options.SessionDuration)

	switch {
	case r.AccessKey != "":
		key, identity, err := access.ExchangeAccessKey(c, r.AccessKey, expires)
		if err != nil {
			return nil, err
		}

		setAuthCookie(c, key, expires)

		if a.t != nil {
			if err := a.t.Enqueue(analytics.Track{Event: "infra.login.exchange", UserId: identity.ID.String()}); err != nil {
				logging.S.Debug(err)
			}
		}

		return &api.LoginResponse{PolymorphicID: identity.PolyID(), Name: identity.Name, AccessKey: key, Expires: api.Time(expires)}, nil
	case r.PasswordCredentials != nil:
		key, user, requiresUpdate, err := access.LoginWithUserCredential(c, r.PasswordCredentials.Email, r.PasswordCredentials.Password, expires)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", internal.ErrUnauthorized, err.Error())
		}

		setAuthCookie(c, key, expires)

		a.t.Event(c, "login", Properties{"method": "credentials"})

		return &api.LoginResponse{PolymorphicID: user.PolyID(), Name: user.Name, AccessKey: key, Expires: api.Time(expires), PasswordUpdateRequired: requiresUpdate}, nil
	case r.OIDC != nil:
		provider, err := access.GetProvider(c, r.OIDC.ProviderID)
		if err != nil {
			return nil, err
		}

		oidc, err := a.providerClient(c, provider, r.OIDC.RedirectURL)
		if err != nil {
			return nil, err
		}

		user, key, err := access.ExchangeAuthCodeForAccessKey(c, r.OIDC.Code, provider, oidc, expires, r.OIDC.RedirectURL)
		if err != nil {
			return nil, err
		}

		setAuthCookie(c, key, expires)

		a.t.Event(c, "login", Properties{"method": "oidc"})

		return &api.LoginResponse{PolymorphicID: user.PolyID(), Name: user.Name, AccessKey: key, Expires: api.Time(expires)}, nil
	}

	return nil, api.ErrBadRequest
}

func (a *API) Logout(c *gin.Context, r *api.EmptyRequest) (*api.EmptyResponse, error) {
	err := access.DeleteRequestAccessKey(c)
	if err != nil {
		return nil, err
	}

	deleteAuthCookie(c)

	a.t.Event(c, "logout")

	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.Version}, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func (a *API) UpdateIdentityInfoFromProvider(c *gin.Context) error {
	user := access.CurrentIdentity(c)
	if user == nil {
		return nil
	}

	providerUser, err := access.RetrieveUserProviderTokens(c)
	if err != nil {
		return err
	}

	provider, err := access.GetProvider(c, providerUser.ProviderID)
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	if provider.Name == models.InternalInfraProviderName {
		return nil
	}

	oidc, err := a.providerClient(c, provider, providerUser.RedirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	// check if the access token needs to be refreshed
	newAccessToken, newExpiry, err := oidc.RefreshAccessToken(providerUser)
	if err != nil {
		return fmt.Errorf("refresh provider access: %w", err)
	}

	if newAccessToken != string(providerUser.AccessToken) {
		logging.S.Debugf("access token for user at provider %s was refreshed", providerUser.ProviderID)

		providerUser.AccessToken = models.EncryptedAtRest(newAccessToken)
		providerUser.ExpiresAt = *newExpiry

		if err := access.UpdateProviderUser(c, providerUser); err != nil {
			return fmt.Errorf("update access token before JWT: %w", err)
		}
	}

	// get current identity provider groups
	info, err := oidc.GetUserInfo(providerUser)
	if err != nil {
		if errors.Is(err, internal.ErrForbidden) {
			err := access.DeleteAllIdentityAccessKeys(c)
			if err != nil {
				logging.S.Errorf("failed to revoke invalid user session: %s", err)
			}

			deleteAuthCookie(c)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return fmt.Errorf("update user info: %w", err)
	}

	return access.UpdateUserInfoFromProvider(c, info, user, provider)
}

func (a *API) providerClient(c *gin.Context, provider *models.Provider, redirectURL string) (authn.OIDC, error) {
	if val, ok := c.Get("oidc"); ok {
		// oidc is added to the context during unit tests
		oidc, _ := val.(authn.OIDC)
		return oidc, nil
	}

	clientSecret, err := secrets.GetSecret(string(provider.ClientSecret), a.server.secrets)
	if err != nil {
		logging.S.Debugf("could not get client secret: %s", err)
		return nil, fmt.Errorf("error loading provider client")
	}

	return authn.NewOIDC(provider.URL, provider.ClientID, clientSecret, redirectURL), nil
}
