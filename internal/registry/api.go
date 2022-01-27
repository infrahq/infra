package registry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gopkg.in/segmentio/analytics-go.v3"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/models"
)

type API struct {
	t        *Telemetry
	registry *Registry
}

func NewAPIMux(reg *Registry, router *gin.RouterGroup) {
	a := API{
		t:        reg.tel,
		registry: reg,
	}

	a.registerRoutes(router)
}

func sendAPIError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	message := "internal server error" // don't leak any info by default

	switch {
	case errors.Is(err, internal.ErrUnauthorized):
		code = http.StatusUnauthorized
		message = "unauthorized"
	case errors.Is(err, internal.ErrForbidden):
		code = http.StatusForbidden
		message = "forbidden"
	case errors.Is(err, internal.ErrDuplicate):
		code = http.StatusConflict
		message = err.Error()
	case errors.Is(err, internal.ErrNotFound):
		code = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, internal.ErrBadRequest):
		code = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, (*validator.InvalidValidationError)(nil)):
		code = http.StatusBadRequest
		message = err.Error()
	}

	logging.Logger(c).Errorw(err.Error(), "statusCode", code)

	c.JSON(code, &api.Error{
		Code:    int32(code),
		Message: message,
	})
	c.Abort()
}

func (a *API) ListUsers(c *gin.Context, r *api.ListUsersRequest) ([]api.User, error) {
	users, err := access.ListUsers(c, r.Email)
	if err != nil {
		return nil, err
	}

	results := make([]api.User, len(users))
	for i, u := range users {
		results[i] = u.ToAPI()
	}

	return results, nil
}

func (a *API) GetUser(c *gin.Context, r *api.Resource) (*api.User, error) {
	user, err := access.GetUser(c, r.ID)
	if err != nil {
		return nil, err
	}

	resp := user.ToAPI()
	return &resp, nil
}

func (a *API) ListGroups(c *gin.Context, r *api.ListGroupsRequest) ([]api.Group, error) {
	groups, err := access.ListGroups(c, r.GroupName)
	if err != nil {
		return nil, err
	}

	results := make([]api.Group, len(groups))
	for i, g := range groups {
		results[i] = g.ToAPI()
	}

	return results, nil
}

func (a *API) GetGroup(c *gin.Context, r *api.Resource) (*api.Group, error) {
	group, err := access.GetGroup(c, r.ID)
	if err != nil {
		return nil, err
	}

	resp := group.ToAPI()
	return &resp, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context, r *api.ListProvidersRequest) ([]api.Provider, error) {
	providers, err := access.ListProviders(c, models.ProviderKind(r.ProviderKind), r.Domain)
	if err != nil {
		return nil, err
	}

	results := make([]api.Provider, len(providers))
	for i, p := range providers {
		results[i] = p.ToAPI()
	}

	return results, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context, r *api.Resource) (*api.Provider, error) {
	provider, err := access.GetProvider(c, r.ID)
	if err != nil {
		return nil, err
	}

	result := provider.ToAPI()
	return &result, nil
}

func (a *API) CreateProvider(c *gin.Context, r *api.CreateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Kind:         models.ProviderKind(r.Kind),
		Domain:       r.Domain,
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	provider, err := access.CreateProvider(c, provider)
	if err != nil {
		return nil, err
	}

	result := provider.ToAPI()
	return &result, nil
}

func (a *API) UpdateProvider(c *gin.Context, r *api.UpdateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Model: models.Model{
			ID: r.ID,
		},
		Kind:         models.ProviderKind(r.Kind),
		Domain:       r.Domain,
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	provider, err := access.UpdateProvider(c, r.ID, provider)
	if err != nil {
		return nil, err
	}

	result := provider.ToAPI()
	return &result, nil
}

func (a *API) DeleteProvider(c *gin.Context, r *api.Resource) error {
	return access.DeleteProvider(c, r.ID)
}

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) ([]api.Destination, error) {
	destinations, err := access.ListDestinations(c, string(r.Kind), r.NodeID, r.Name, r.Labels)
	if err != nil {
		return nil, err
	}

	results := make([]api.Destination, len(destinations))
	for i, d := range destinations {
		results[i] = *d.ToAPI()
	}

	return results, nil
}

func (a *API) GetDestination(c *gin.Context, r *api.Resource) (*api.Destination, error) {
	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		return nil, err
	}

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(c *gin.Context, r *api.CreateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{}
	if err := destination.FromCreateAPI(r); err != nil {
		return nil, err
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	sync := func(db *gorm.DB) error {
		return importGrantMappings(db, a.registry.options.Users, a.registry.options.Groups)
	}
	if err := access.SyncGrants(c, sync); err != nil {
		return nil, fmt.Errorf("sync grants destination create: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{Model: models.Model{ID: r.ID}}
	if err := destination.FromUpdateAPI(r); err != nil {
		return nil, err
	}

	if err := access.UpdateDestination(c, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	sync := func(db *gorm.DB) error {
		return importGrantMappings(db, a.registry.options.Users, a.registry.options.Groups)
	}
	if err := access.SyncGrants(c, sync); err != nil {
		return nil, fmt.Errorf("sync grants destination update: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) error {
	return access.DeleteDestination(c, r.ID)
}

func (a *API) ListAPITokens(c *gin.Context, r *api.ListAPITokensRequest) ([]api.InfraAPIToken, error) {
	keyTuples, err := access.ListAPITokens(c, r.KeyName)
	if err != nil {
		return nil, err
	}

	results := make([]api.InfraAPIToken, len(keyTuples))

	for i, k := range keyTuples {
		results[i] = *(k.ToAPI())
	}

	return results, nil
}

func (a *API) DeleteAPIToken(c *gin.Context, r *api.Resource) error {
	return access.RevokeAPIToken(c, r.ID)
}

func (a *API) CreateAPIToken(c *gin.Context, r *api.InfraAPITokenCreateRequest) (*api.InfraAPITokenCreateResponse, error) {
	apiToken := &models.APIToken{}
	if err := apiToken.FromAPI(r, DefaultSessionDuration); err != nil {
		return nil, err
	}

	tkn, err := access.IssueAPIToken(c, apiToken)
	if err != nil {
		return nil, err
	}

	return apiToken.ToAPICreateResponse(tkn), nil
}

func (a *API) ListGrants(c *gin.Context, r *api.ListGrantsRequest) ([]api.Grant, error) {
	grants, err := access.ListGrants(c, models.GrantKind(r.GrantKind), r.DestinationID)
	if err != nil {
		return nil, err
	}

	results := make([]api.Grant, 0)
	for _, r := range grants {
		results = append(results, r.ToAPI())
	}

	return results, nil
}

func (a *API) GetGrant(c *gin.Context, r *api.Resource) (*api.Grant, error) {
	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		return nil, err
	}

	result := grant.ToAPI()
	return &result, nil
}

func (a *API) CreateToken(c *gin.Context, r *api.TokenRequest) (*api.Token, error) {
	// sync the user information before doing this sensitive action
	err := a.updateUserInfo(c)
	if err != nil {
		return nil, err
	}

	token, expiry, err := access.IssueJWT(c, r.Destination)
	if err != nil {
		return nil, err
	}

	return &api.Token{Token: token, Expires: expiry.Unix()}, nil
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	if r.Okta != nil {
		providers, err := access.ListProviders(c, "okta", r.Okta.Domain)
		if err != nil {
			return nil, err
		}

		if len(providers) == 0 {
			return nil, fmt.Errorf("%w: no such provider", internal.ErrBadRequest)
		}

		provider := &providers[0] // TODO: should probably check all providers, not the first one.

		oidc, err := a.providerClient(c, provider)
		if err != nil {
			return nil, err
		}

		user, token, err := access.ExchangeAuthCodeForSessionToken(c, r.Okta.Code, provider, oidc, a.registry.options.SessionDuration)
		if err != nil {
			return nil, err
		}

		setAuthCookie(c, token, a.registry.options.SessionDuration)

		if a.t != nil {
			if err := a.t.Enqueue(analytics.Track{Event: "infra.login", UserId: user.ID.String()}); err != nil {
				logging.S.Debug(err)
			}
		}

		return &api.LoginResponse{Name: user.Email, Token: token}, nil
	}

	return nil, fmt.Errorf("%w: invalid login request", internal.ErrBadRequest)
}

func (a *API) Logout(c *gin.Context, r *api.EmptyRequest) (*api.EmptyResponse, error) {
	token, err := access.RevokeToken(c)
	if err != nil {
		return nil, err
	}

	deleteAuthCookie(c)

	if a.t != nil {
		if err := a.t.Enqueue(analytics.Track{Event: "infra.logout", UserId: token.UserID.String()}); err != nil {
			logging.S.Debug(err)
		}
	}

	return nil, nil
}

func (a *API) Version(c *gin.Context, r *api.EmptyRequest) (*api.Version, error) {
	return &api.Version{Version: internal.Version}, nil
}

// updateUserInfo calls the identity provider used to authenticate this user session to update their current information
func (a *API) updateUserInfo(c *gin.Context) error {
	providerTokens, err := access.RetrieveUserProviderTokens(c)
	if err != nil {
		return err
	}

	provider, err := access.GetProvider(c, providerTokens.ProviderID)
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	user := access.CurrentUser(c)
	if user == nil {
		return fmt.Errorf("user not found in context for info update")
	}

	if provider.Kind == models.ProviderKindOkta {
		oidc, err := a.providerClient(c, provider)
		if err != nil {
			return fmt.Errorf("update provider client: %w", err)
		}

		// check if the access token needs to be refreshed
		newAccessToken, newExpiry, err := oidc.RefreshAccessToken(providerTokens)
		if err != nil {
			return fmt.Errorf("refresh provider access: %w", err)
		}

		if newAccessToken != string(providerTokens.AccessToken) {
			logging.S.Debugf("access token for user at provider %s was refreshed", providerTokens.ProviderID)

			providerTokens.AccessToken = models.EncryptedAtRest(newAccessToken)
			providerTokens.Expiry = *newExpiry

			if err := access.UpdateProviderToken(c, providerTokens); err != nil {
				return fmt.Errorf("update access token before JWT: %w", err)
			}
		}

		// get current identity provider groups
		info, err := oidc.GetUserInfo(providerTokens)
		if err != nil {
			if errors.Is(err, internal.ErrForbidden) {
				_, err := access.RevokeToken(c)
				if err != nil {
					logging.S.Errorf("failed to revoke invalid user session: %w", err)
				}
				deleteAuthCookie(c)
			}
			return fmt.Errorf("update user info: %w", err)
		}

		return access.UpdateUserInfo(c, info, user, provider)
	}

	return fmt.Errorf("unknown provider kind for user info update")
}

func (a *API) providerClient(c *gin.Context, provider *models.Provider) (authn.OIDC, error) {
	if val, ok := c.Get("oidc"); ok {
		// oidc is added to the context during unit tests
		oidc, _ := val.(authn.OIDC)
		return oidc, nil
	}

	clientSecret, err := a.registry.GetSecret(string(provider.ClientSecret))
	if err != nil {
		logging.S.Debugf("could not get client secret: %w", err)
		return nil, fmt.Errorf("error loading provider client")
	}

	return authn.NewOIDC(provider.Domain, provider.ClientID, clientSecret), nil
}
