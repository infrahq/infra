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

		authorized.GET("/groups", a.ListGroups)
		authorized.GET("/groups/:id", a.GetGroup)

		authorized.GET("/grants", a.ListGrants)
		authorized.GET("/grants/:id", a.GetGrant)

		authorized.GET("/destinations", a.ListDestinations)
		authorized.GET("/destinations/:id", a.GetDestination)
		authorized.POST("/destinations", a.CreateDestination)

		authorized.GET("/api-keys", a.ListAPIKeys)
		authorized.POST("/api-keys", a.CreateAPIKey)
		authorized.DELETE("/api-keys/:id", a.DeleteAPIKey)

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
	userID := c.Param("id")
	if userID == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid user ID"))
		return
	}

	user, err := access.GetUser(c, userID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := user.ToAPI()

	c.JSON(http.StatusOK, result)
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
	groupID := c.Param("id")
	if groupID == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid group ID"))
		return
	}

	group, err := access.GetGroup(c, groupID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := group.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) ListProviders(c *gin.Context) {
	// caution: this endpoint is unauthenticated, do not return sensitive info
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

func (a *API) GetProvider(c *gin.Context) {
	// caution: this endpoint is unauthenticated, do not return sensitive info
	providerID := c.Param("id")
	if providerID == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid provider ID"))
		return
	}

	provider, err := access.GetProvider(c, providerID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := provider.ToAPI()

	c.JSON(http.StatusOK, result)
}

func (a *API) ListDestinations(c *gin.Context) {
	destinationName := c.Request.URL.Query().Get("name")
	destinationKind := c.Request.URL.Query().Get("kind")

	destinations, err := access.ListDestinations(c, destinationName, destinationKind)
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
	destinationID := c.Param("id")
	if destinationID == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid destination ID"))
		return
	}

	destination, err := access.GetDestination(c, destinationID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := destination.ToAPI()
	c.JSON(http.StatusOK, result)
}

func (a *API) CreateDestination(c *gin.Context) {
	var body api.DestinationCreateRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(body); err != nil {
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

func (a *API) ListAPIKeys(c *gin.Context) {
	keyName := c.Request.URL.Query().Get("name")

	keys, err := access.ListAPIKeys(c, keyName)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	results := make([]api.InfraAPIKey, 0)

	for _, k := range keys {
		results = append(results, k.ToAPI())
	}

	c.JSON(http.StatusOK, results)
}

func (a *API) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid API key ID"))
		return
	}

	if err := access.RevokeAPIKey(c, id); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

func (a *API) CreateAPIKey(c *gin.Context) {
	var body api.InfraAPIKeyCreateRequest
	if err := c.BindJSON(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	if err := validate.Struct(body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	apiKey := &models.APIKey{}
	if err := apiKey.FromAPI(&body); err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	apiKey, err := access.IssueAPIKey(c, apiKey)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := apiKey.ToAPICreateResponse()
	c.JSON(http.StatusCreated, result)
}

func (a *API) ListGrants(c *gin.Context) {
	grantKind := c.Request.URL.Query().Get("kind")
	destinationID := c.Request.URL.Query().Get("destination")

	grants, err := access.ListGrants(c, grantKind, destinationID)
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
	grantID := c.Param("id")
	if grantID == "" {
		sendAPIError(c, http.StatusBadRequest, fmt.Errorf("invalid grant ID"))
		return
	}

	grant, err := access.GetGrant(c, grantID)
	if err != nil {
		sendAPIError(c, http.StatusBadRequest, err)
		return
	}

	result := grant.ToAPI()

	c.JSON(http.StatusOK, result)
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

	if err := a.t.Enqueue(analytics.Track{Event: "infra.logout", UserId: token.UserID.String()}); err != nil {
		logging.S.Debug(err)
	}

	c.Status(http.StatusOK)
}

func (a *API) Version(c *gin.Context) {
	c.JSON(http.StatusOK, api.Version{Version: internal.Version})
}
