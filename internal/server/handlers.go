package server

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/infrahq/secrets"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

type API struct {
	t          *Telemetry
	server     *Server
	migrations []apiMigration
	openAPIDoc openapi3.T
}

func (a *API) ListUsers(c *gin.Context, r *api.ListUsersRequest) (*api.ListResponse[api.User], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	users, err := access.ListIdentities(c, r.Name, r.Group, r.IDs, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(users, models.PaginationToResponse(p), func(identity models.Identity) api.User {
		return *identity.ToAPI()
	})

	return result, nil
}

func (a *API) GetUser(c *gin.Context, r *api.GetUserRequest) (*api.User, error) {
	if r.ID.IsSelf {
		iden := access.AuthenticatedIdentity(c)
		if iden == nil {
			return nil, internal.ErrUnauthorized
		}
		r.ID.ID = iden.ID
	}
	identity, err := access.GetIdentity(c, r.ID.ID)
	if err != nil {
		return nil, err
	}

	return identity.ToAPI(), nil
}

// CreateUser creates a user with the Infra provider
func (a *API) CreateUser(c *gin.Context, r *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	user := &models.Identity{Name: r.Name}
	infraProvider := access.InfraProvider(c)

	// infra identity creation should be attempted even if an identity is already known
	identities, err := access.ListIdentities(c, user.Name, 0, nil, &models.Pagination{Limit: 2})
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}

	switch len(identities) {
	case 0:
		if err := access.CreateIdentity(c, user); err != nil {
			return nil, fmt.Errorf("create identity: %w", err)
		}
	case 1:
		user.ID = identities[0].ID
	default:
		logging.S.Errorf("Multiple identites match name %q. DB is missing unique index on user names", r.Name)
		return nil, fmt.Errorf("multiple identities match specified name") // should not happen
	}

	resp := &api.CreateUserResponse{
		ID:   user.ID,
		Name: user.Name,
	}

	_, err = access.CreateProviderUser(c, infraProvider, user)
	if err != nil {
		return nil, fmt.Errorf("creating provider user: %w", err)
	}

	// Always create a temporary password for infra users.
	tmpPassword, err := access.CreateCredential(c, *user)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	resp.OneTimePassword = tmpPassword

	return resp, nil
}

func (a *API) UpdateUser(c *gin.Context, r *api.UpdateUserRequest) (*api.User, error) {
	// right now this endpoint can only update a user's credentials, so get the user identity
	identity, err := access.GetIdentity(c, r.ID)
	if err != nil {
		return nil, err
	}

	err = access.UpdateCredential(c, identity, r.Password)
	if err != nil {
		return nil, err
	}

	// if the user is an admin, we could be required to create the infra user, so create the provider_user if it's missing.
	_, _ = access.CreateProviderUser(c, access.InfraProvider(c), identity)

	return identity.ToAPI(), nil
}

func (a *API) DeleteUser(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteIdentity(c, r.ID)
}

// TODO: remove after deprecation period
func (a *API) deprecatedListUserGroups(c *gin.Context, r *api.Resource) (*api.ListResponse[api.Group], error) {
	return a.ListGroups(c, &api.ListGroupsRequest{UserID: r.ID})
}

func (a *API) ListGroups(c *gin.Context, r *api.ListGroupsRequest) (*api.ListResponse[api.Group], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	groups, err := access.ListGroups(c, r.Name, r.UserID, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(groups, models.PaginationToResponse(p), func(group models.Group) api.Group {
		return *group.ToAPI()
	})

	return result, nil
}

func (a *API) GetGroup(c *gin.Context, r *api.Resource) (*api.Group, error) {
	group, err := access.GetGroup(c, r.ID)
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) CreateGroup(c *gin.Context, r *api.CreateGroupRequest) (*api.Group, error) {
	group := &models.Group{
		Name: r.Name,
	}

	authIdent := access.AuthenticatedIdentity(c)
	if authIdent != nil {
		group.CreatedBy = authIdent.ID
	}

	err := access.CreateGroup(c, group)
	if err != nil {
		return nil, err
	}

	return group.ToAPI(), nil
}

func (a *API) DeleteGroup(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteGroup(c, r.ID)
}

func (a *API) UpdateUsersInGroup(c *gin.Context, r *api.UpdateUsersInGroupRequest) (*api.EmptyResponse, error) {
	return nil, access.UpdateUsersInGroup(c, r.GroupID, r.UserIDsToAdd, r.UserIDsToRemove)
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context, r *api.ListProvidersRequest) (*api.ListResponse[api.Provider], error) {
	exclude := []string{models.InternalInfraProviderName}
	p := models.RequestToPagination(r.PaginationRequest)
	providers, err := access.ListProviders(c, r.Name, exclude, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(providers, models.PaginationToResponse(p), func(provider models.Provider) api.Provider {
		return *provider.ToAPI()
	})

	return result, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context, r *api.Resource) (*api.Provider, error) {
	provider, err := access.GetProvider(c, r.ID)
	if err != nil {
		return nil, err
	}

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

	kind, err := models.ParseProviderKind(r.Kind)
	if err != nil {
		return nil, err
	}
	provider.Kind = kind

	if err := a.validateProvider(c, provider); err != nil {
		return nil, err
	}

	if err := access.CreateProvider(c, provider); err != nil {
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

	kind, err := models.ParseProviderKind(r.Kind)
	if err != nil {
		return nil, err
	}
	provider.Kind = kind

	if err := a.validateProvider(c, provider); err != nil {
		return nil, err
	}

	if err := access.SaveProvider(c, provider); err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

func (a *API) DeleteProvider(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteProvider(c, r.ID)
}

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	p := models.RequestToPagination(r.PaginationRequest)
	destinations, err := access.ListDestinations(c, r.UniqueID, r.Name, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(destinations, models.PaginationToResponse(p), func(destination models.Destination) api.Destination {
		return *destination.ToAPI()
	})

	return result, nil
}

func (a *API) GetDestination(c *gin.Context, r *api.Resource) (*api.Destination, error) {
	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		return nil, err
	}

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(c *gin.Context, r *api.CreateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

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
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
	}

	if err := access.SaveDestination(c, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteDestination(c, r.ID)
}

func (a *API) CreateToken(c *gin.Context, r *api.EmptyRequest) (*api.CreateTokenResponse, error) {
	if access.AuthenticatedIdentity(c) != nil {
		err := a.UpdateIdentityInfoFromProvider(c)
		if err != nil {
			// TODO: why would this fail? seems like this should be a 5xx error
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

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) (*api.ListResponse[api.Grant], error) {
	var subject uid.PolymorphicID
	p := models.RequestToPagination(r.PaginationRequest)
	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	grants, err := access.ListGrants(c, subject, r.Resource, r.Privilege, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(grants, models.PaginationToResponse(p), func(grant models.Grant) api.Grant {
		return *grant.ToAPI()
	})

	return result, nil
}

// TODO: remove after deprecation period
func (a *API) deprecatedListUserGrants(c *gin.Context, r *api.Resource) (*api.ListResponse[api.Grant], error) {
	return a.ListGrants(c, &api.ListGrantsRequest{User: r.ID})
}

// TODO: remove after deprecation period
func (a *API) deprecatedListGroupGrants(c *gin.Context, r *api.Resource) (*api.ListResponse[api.Grant], error) {
	return a.ListGrants(c, &api.ListGrantsRequest{Group: r.ID})
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) CreateGrant(c *gin.Context, r *api.CreateGrantRequest) (*api.Grant, error) {
	var subject uid.PolymorphicID

	switch {
	case r.User != 0:
		subject = uid.NewIdentityPolymorphicID(r.User)
	case r.Group != 0:
		subject = uid.NewGroupPolymorphicID(r.Group)
	}

	grant := &models.Grant{
		Subject:   subject,
		Resource:  r.Resource,
		Privilege: r.Privilege,
	}

	err := access.CreateGrant(c, grant)
	if err != nil {
		return nil, err
	}

	return grant.ToAPI(), nil
}

func (a *API) DeleteGrant(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	if grant.Resource == access.ResourceInfraAPI && grant.Privilege == models.InfraAdminRole {
		infraAdminGrants, err := access.ListGrants(c, "", grant.Resource, grant.Privilege, &models.Pagination{})
		if err != nil {
			return nil, err
		}

		if len(infraAdminGrants) == 1 {
			return nil, fmt.Errorf("%w: cannot remove the last infra admin", internal.ErrBadRequest)
		}
	}

	return nil, access.DeleteGrant(c, r.ID)
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

	if r.Name == "" {
		// #1825: remove, this is for migration
		r.Name = r.Email
	}

	identity, err := access.Signup(c, r.Name, r.Password)
	if err != nil {
		return nil, err
	}

	a.t.User(identity.ID.String(), r.Name)
	a.t.Event("signup", identity.ID.String(), Properties{})

	return identity.ToAPI(), nil
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	var loginMethod authn.LoginMethod

	expires := time.Now().UTC().Add(a.server.options.SessionDuration)

	switch {
	case r.AccessKey != "":
		loginMethod = authn.NewKeyExchangeAuthentication(r.AccessKey, expires)
	case r.PasswordCredentials != nil:
		if r.PasswordCredentials.Name == "" {
			// #1825: remove, this is for migration
			r.PasswordCredentials.Name = r.PasswordCredentials.Email
		}
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
		logging.S.Debug(err)
		// all other failures from login should result in an unauthorized response
		return nil, internal.ErrUnauthorized
	}

	setAuthCookie(c, bearer, expires)

	a.t.Event("login", key.IssuedFor.String(), Properties{"method": loginMethod.Name()})

	return &api.LoginResponse{UserID: key.IssuedFor, Name: key.IssuedForIdentity.Name, AccessKey: bearer, Expires: api.Time(expires), PasswordUpdateRequired: requiresUpdate}, nil
}

func (a *API) Logout(c *gin.Context, r *api.EmptyRequest) (*api.EmptyResponse, error) {
	err := access.DeleteRequestAccessKey(c)
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
	provider, redirectURL, err := access.GetContextProviderIdentity(c)
	if err != nil {
		return err
	}

	if provider.Name == models.InternalInfraProviderName {
		return nil
	}

	oidc, err := a.providerClient(c, provider, redirectURL)
	if err != nil {
		return fmt.Errorf("update provider client: %w", err)
	}

	return access.UpdateIdentityInfoFromProvider(c, oidc)
}

// validateProvider checks that a provider being modified is valid
func (a *API) validateProvider(c *gin.Context, provider *models.Provider) error {
	oidc, err := a.providerClient(c, provider, "") // redirect URL is not used during validation
	if err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	return oidc.Validate(c.Request.Context())
}

func (a *API) providerClient(c *gin.Context, provider *models.Provider, redirectURL string) (providers.OIDC, error) {
	if val, ok := c.Get("oidc"); ok {
		// oidc is added to the context during unit tests
		oidc, _ := val.(providers.OIDC)
		return oidc, nil
	}

	clientSecret, err := secrets.GetSecret(string(provider.ClientSecret), a.server.secrets)
	if err != nil {
		logging.S.Debugf("could not get client secret: %s", err)
		return nil, fmt.Errorf("client secret not found")
	}

	return providers.NewOIDC(*provider, clientSecret, redirectURL), nil
}
