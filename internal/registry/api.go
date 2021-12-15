package registry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

type Resource struct {
	ID string `uri:"id" binding:"required,uuid"`
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
		authorized.GET("/users", a.ListUsers)
		authorized.GET("/users/:id", a.GetUser)
		authorized.GET("/users/:id/grants", a.GetUserGrants)

		authorized.GET("/groups", a.ListGroups)
		authorized.GET("/groups/:id", a.GetGroup)
		authorized.GET("/groups/:id/grants", a.GetGroupGrants)

		authorized.GET("/grants", a.ListGrants)
		authorized.GET("/grants/:id", a.GetGrant)
		authorized.POST("/grants", a.CreateGrant)
		authorized.PUT("/grants/:id", a.UpdateGrant)
		authorized.DELETE("/grants/:id", a.DeleteGrant)

		authorized.POST("/providers", a.CreateProvider)
		authorized.PUT("/providers/:id", a.UpdateProvider)
		authorized.DELETE("/providers/:id", a.DeleteProvider)

		authorized.GET("/destinations", a.ListDestinations)
		authorized.GET("/destinations/:id", a.GetDestination)
		authorized.GET("/destinations/:id/grants", a.GetDestinationGrants)
		authorized.POST("/destinations", a.CreateDestination)
		authorized.PUT("/destinations/:id", a.UpdateDestination)
		authorized.DELETE("/destinations/:id", a.DeleteDestination)

		authorized.GET("/api-tokens", a.ListAPITokens)
		authorized.POST("/api-tokens", a.CreateAPIToken)
		authorized.DELETE("/api-tokens/:id", a.DeleteAPIToken)

		authorized.POST("/tokens", a.CreateToken)
		authorized.POST("/logout", a.Logout)
	}

	// these endpoints are left unauthenticated so that infra login can see what the providers are that are available
	unauthorized := router.Group("/")

	{
		unauthorized.GET("/providers", a.ListProviders)
		unauthorized.GET("/providers/:id", a.GetProvider)

		unauthorized.POST("/login", a.Login)
		unauthorized.GET("/version", a.Version)
	}
}

func sendAPIError(c *gin.Context, code int, err error) {
	message := err.Error()

	switch {
	case errors.Is(err, internal.ErrExpired):
		fallthrough
	case errors.Is(err, internal.ErrInvalid):
		code = http.StatusUnauthorized
		message = "unauthorized"
	case errors.Is(err, internal.ErrForbidden):
		code = http.StatusForbidden
		message = "forbidden"
	case errors.Is(err, internal.ErrDuplicate):
		code = http.StatusConflict
	case errors.Is(err, internal.ErrNotFound):
		code = http.StatusNotFound
	}

	c.JSON(code, &api.Error{
		Code:    int32(code),
		Message: message,
	})
	c.Abort()
}

func (a *API) ListUsers(c *gin.Context) {
	userEmail := c.Request.URL.Query().Get("email")

	users, err := access.ListUsers(c, userEmail)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.User, 0)
	for _, u := range users {
		results = append(results, u.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetUser(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	user, err := access.GetUser(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := user.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) GetUserGrants(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	user, err := access.GetUser(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	grants, err := access.ListGrants(c, user, nil, nil)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	results := make([]api.Grant, 0)
	for _, g := range grants {
		results = append(results, g.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) ListGroups(c *gin.Context) {
	groupName := c.Request.URL.Query().Get("name")

	groups, err := access.ListGroups(c, groupName)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.Group, 0)
	for _, g := range groups {
		results = append(results, g.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetGroup(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	group, err := access.GetGroup(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := group.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) GetGroupGrants(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	group, err := access.GetGroup(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	grants, err := access.ListGrants(c, nil, group, nil)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	results := make([]api.Grant, 0)
	for _, g := range grants {
		results = append(results, g.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context) {
	providerKind := c.Request.URL.Query().Get("kind")
	providerDomain := c.Request.URL.Query().Get("domain")

	providers, err := access.ListProviders(c, providerKind, providerDomain)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.Provider, 0)
	for _, p := range providers {
		results = append(results, p.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	provider, err := access.GetProvider(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := provider.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) CreateProvider(c *gin.Context) {
	var body api.ProviderRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	provider := &models.Provider{}
	if err := provider.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	provider, err := access.CreateProvider(c, provider)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	result := provider.ToAPI()
	c.JSON(http.StatusCreated, result)
}

func (a *API) UpdateProvider(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	var body api.ProviderRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	provider, err := models.NewProvider(r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := provider.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	provider, err = access.UpdateProvider(c, r.ID, provider)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	result := provider.ToAPI()
	c.JSON(http.StatusOK, result)
}

func (a *API) DeleteProvider(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := access.DeleteProvider(c, r.ID); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

func (a *API) ListDestinations(c *gin.Context) {
	var query api.Destination
	if err := c.ShouldBindQuery(&query); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destinations, err := access.ListDestinations(c, string(query.Kind), query.NodeID, query.Name, query.Labels)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.Destination, 0)
	for _, d := range destinations {
		results = append(results, d.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetDestination(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := destination.ToAPI()
	c.JSON(http.StatusOK, result)
}

func (a *API) GetDestinationGrants(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination, err := access.GetDestination(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grants, err := access.ListGrants(c, nil, nil, destination)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.Grant, 0)
	for _, grant := range grants {
		results = append(results, grant.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) CreateDestination(c *gin.Context) {
	var body api.DestinationRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination := &models.Destination{}
	if err := destination.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination, err := access.CreateDestination(c, destination)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := destination.ToAPI()
	c.JSON(http.StatusCreated, result)
}

func (a *API) UpdateDestination(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	var body api.DestinationRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination, err := models.NewDestination(r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := destination.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	destination, err = access.UpdateDestination(c, r.ID, destination)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := destination.ToAPI()
	c.JSON(http.StatusOK, result)
}

func (a *API) DeleteDestination(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := access.DeleteDestination(c, r.ID); err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

func (a *API) ListAPITokens(c *gin.Context) {
	keyName := c.Request.URL.Query().Get("name")

	keyTuples, err := access.ListAPITokens(c, keyName)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.InfraAPIToken, 0)

	for _, k := range keyTuples {
		key := k.ToAPI()
		results = append(results, *key)
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) DeleteAPIToken(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid API token ID"))
		return
	}

	if err := access.RevokeAPIToken(c, id); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

func (a *API) CreateAPIToken(c *gin.Context) {
	var body api.InfraAPITokenCreateRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	apiToken := &models.APIToken{}
	if err := apiToken.FromAPI(&body, DefaultSessionDuration); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	apiToken, tkn, err := access.IssueAPIToken(c, apiToken)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, apiToken.ToAPICreateResponse(tkn))
}

func (a *API) ListGrants(c *gin.Context) {
	grants, err := access.ListGrants(c, nil, nil, nil)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.Grant, 0)
	for _, r := range grants {
		results = append(results, r.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) GetGrant(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grant, err := access.GetGrant(c, r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := grant.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) CreateGrant(c *gin.Context) {
	var body api.GrantRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grant := &models.Grant{}
	if err := grant.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grant, err := access.CreateGrant(c, grant)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	result := grant.ToAPI()
	c.JSON(http.StatusCreated, result)
}

func (a *API) UpdateGrant(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	var body api.GrantRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grant, err := models.NewGrant(r.ID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := grant.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	grant, err = access.UpdateGrant(c, r.ID, grant)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, err)
		return
	}

	result := grant.ToAPI()
	c.JSON(http.StatusOK, result)
}

func (a *API) DeleteGrant(c *gin.Context) {
	var r Resource
	if err := c.BindUri(&r); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := access.DeleteGrant(c, r.ID); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

func (a *API) CreateToken(c *gin.Context) {
	var body api.TokenRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	token, expiry, err := access.IssueJWT(c, body.Destination)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, api.Token{Token: token, Expires: expiry.Unix()})
}

func (a *API) Login(c *gin.Context) {
	var body api.LoginRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	var email string

	switch {
	case body.Okta != nil:
		providers, err := access.ListProviders(c, "okta", body.Okta.Domain)
		if err != nil {
			sendAPIError(c, http.StatusBadRequest, err)
			return
		}

		if len(providers) == 0 {
			sendAPIError(c, http.StatusBadRequest, err)
			return
		}

		provider := providers[0]

		clientSecret, err := a.registry.GetSecret(string(provider.ClientSecret))
		if err != nil {
			sendAPIError(c, http.StatusBadRequest, err)
			return
		}

		var okta Okta
		if val, ok := c.Get("okta"); ok {
			okta, _ = val.(Okta)
		} else {
			okta = NewOkta()
		}

		email, err = okta.EmailFromCode(body.Okta.Code, provider.Domain, provider.ClientID, clientSecret)
		if err != nil {
			sendAPIError(c, http.StatusBadRequest, err)
			return
		}

	default:
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid login request"))
		return
	}

	user, token, err := access.IssueUserToken(c, email, a.registry.options.SessionDuration)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	setAuthCookie(c, token.SessionToken(), a.registry.options.SessionDuration)

	if a.t != nil {
		if err := a.t.Enqueue(analytics.Track{Event: "infra.login", UserId: user.ID.String()}); err != nil {
			logging.S.Debug(err)
		}
	}

	c.JSON(http.StatusOK, api.LoginResponse{Name: user.Email, Token: token.SessionToken()})
}

func (a *API) Logout(c *gin.Context) {
	token, err := access.RevokeToken(c)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	deleteAuthCookie(c)

	if a.t != nil {
		if err := a.t.Enqueue(analytics.Track{Event: "infra.logout", UserId: token.UserID.String()}); err != nil {
			logging.S.Debug(err)
		}
	}

	c.Status(http.StatusOK)
}

func (a *API) Version(c *gin.Context) {
	c.JSON(http.StatusOK, api.Version{Version: internal.Version})
}
