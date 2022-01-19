package registry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
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

	router.Use(
		RequestTimeoutMiddleware(),
		MetricsMiddleware(),
		DatabaseMiddleware(reg.db),
	)

	authorized := router.Group("/",
		AuthenticationMiddleware(),
		logging.UserAwareLoggerMiddleware(),
	)

	{
		get(authorized, "/users", a.ListUsers)
		get(authorized, "/users/:id", a.GetUser)

		get(authorized, "/groups", a.ListGroups)
		get(authorized, "/groups/:id", a.GetGroup)

		get(authorized, "/grants", a.ListGrants)
		get(authorized, "/grants/:id", a.GetGrant)

		post(authorized, "/providers", a.CreateProvider)
		put(authorized, "/providers/:id", a.UpdateProvider)
		delete(authorized, "/providers/:id", a.DeleteProvider)

		get(authorized, "/destinations", a.ListDestinations)
		get(authorized, "/destinations/:id", a.GetDestination)
		post(authorized, "/destinations", a.CreateDestination)
		put(authorized, "/destinations/:id", a.UpdateDestination)
		delete(authorized, "/destinations/:id", a.DeleteDestination)

		get(authorized, "/api-tokens", a.ListAPITokens)
		post(authorized, "/api-tokens", a.CreateAPIToken)
		delete(authorized, "/api-tokens/:id", a.DeleteAPIToken)

		post(authorized, "/tokens", a.CreateToken)
		post(authorized, "/logout", a.Logout)
	}

	// these endpoints are left unauthenticated
	unauthorized := router.Group("/")

	{
		get(unauthorized, "/providers", a.ListProviders)
		get(unauthorized, "/providers/:id", a.GetProvider)

		post(unauthorized, "/login", a.Login)
		get(unauthorized, "/version", a.Version)
	}

}

func get[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := *new(Req)
		// TODO: bind uri and query
		c.Bind(req)
		if err := validate.Struct(req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func post[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := *new(Req)
		// TODO: bind uri and query
		c.Bind(req)
		if err := validate.Struct(req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		c.JSON(http.StatusCreated, resp)
	}
}

func put[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := *new(Req)
		// TODO: bind uri and query
		c.Bind(req)
		if err := validate.Struct(req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func delete[Req any](r *gin.RouterGroup, path string, handler ReqHandlerFunc[Req]) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := *new(Req)
		// TODO: bind uri and query
		c.Bind(req)
		if err := validate.Struct(req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
		err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
	}
}

type ReqHandlerFunc[Req any] func(c *gin.Context, req Req) error
type ResHandlerFunc[Res any] func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req Req) (Res, error)

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
	provider := &models.Provider{}
	if err := provider.FromAPI(&r); err != nil {
		return nil, err
	}

	provider, err := access.CreateProvider(c, provider)
	if err != nil {
		return nil, err
	}

	result := provider.ToAPI()
	return &result, nil
}

func (a *API) UpdateProvider(c *gin.Context, r *api.UpdateProviderRequest) (*api.Provider, error) {
	provider := models.NewProvider(r.ID)
	if err := provider.FromAPI(r); err != nil {
		return nil, err
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
	if err := destination.FromAPI(r); err != nil {
		return nil, err
	}

	err := access.CreateDestination(c, destination)
	if err != nil {
		return nil, err
	}

	result := destination.ToAPI()
	return result, nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	destination := &models.Destination{Model: models.Model{ID: r.ID}}
	if err := destination.FromAPI(r); err != nil {
		return nil, err
	}

	if err := access.UpdateDestination(c, destination); err != nil {
		return nil, err
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
	token, expiry, err := access.IssueJWT(c, r.Destination)
	if err != nil {
		return nil, err
	}

	return &api.Token{Token: token, Expires: expiry.Unix()}, nil
}

func (a *API) Login(c *gin.Context, r *api.LoginRequest) (*api.LoginResponse, error) {
	var email string

	switch {
	case r.Okta != nil:
		providers, err := access.ListProviders(c, "okta", r.Okta.Domain)
		if err != nil {
			return nil, err
		}

		if len(providers) == 0 {
			return nil, fmt.Errorf("%w: no such provider", internal.ErrBadRequest)
		}

		provider := providers[0] // TODO: should probably check all providers, not the first one.

		clientSecret, err := a.registry.GetSecret(string(provider.ClientSecret))
		if err != nil {
			return nil, err
		}

		var okta Okta
		if val, ok := c.Get("okta"); ok {
			okta, _ = val.(Okta)
		} else {
			okta = NewOkta()
		}

		email, err = okta.EmailFromCode(r.Okta.Code, provider.Domain, provider.ClientID, clientSecret)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("%w: invalid login request", internal.ErrBadRequest)
	}

	user, token, err := access.IssueUserToken(c, email, a.registry.options.SessionDuration)
	if err != nil {
		return nil, err
	}

	setAuthCookie(c, token.SessionToken(), a.registry.options.SessionDuration)

	if a.t != nil {
		if err := a.t.Enqueue(analytics.Track{Event: "infra.login", UserId: user.ID.String()}); err != nil {
			logging.S.Debug(err)
		}
	}

	return &api.LoginResponse{Name: user.Email, Token: token.SessionToken()}, nil
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
