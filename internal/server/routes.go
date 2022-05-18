package server

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/metrics"
)

// Routes is the return value of GenerateRoutes.
type Routes struct {
	http.Handler
	OpenAPIDocument openapi3.T
}

// GenerateRoutes constructs a http.Handler for the primary http and https servers.
// The handler includes gin middleware, API routes, UI routes, and others.
//
// The returned Routes also include an OpenAPIDocument which can be used to
// generate document about the routes.
//
// The order of routes in this function is important! Gin saves a route along
// with all the middleware that will apply to the route when the
// Router.{GET,POST,etc} method is called.
func (s *Server) GenerateRoutes(promRegistry prometheus.Registerer) Routes {
	a := &API{t: s.tel, server: s}
	router := gin.New()
	router.NoRoute(a.notFoundHandler)

	router.Use(gin.Recovery())
	router.GET("/healthz", healthHandler)

	// This group of middleware will apply to everything, including the UI
	router.Use(
		logging.Middleware(),
		TimeoutMiddleware(1*time.Minute),
	)

	a.addRewrites()
	a.addRedirects()

	// This group of middleware only applies to non-ui routes
	api := router.Group("/",
		metrics.Middleware(promRegistry),
		DatabaseMiddleware(a.server.db), // must be after TimeoutMiddleware to time out db queries.
	)
	api.GET("/.well-known/jwks.json", a.wellKnownJWKsHandler)

	authn := api.Group("/", AuthenticationMiddleware(a))

	get(a, authn, "/api/users", a.ListUsers)
	post(a, authn, "/api/users", a.CreateUser)
	get(a, authn, "/api/users/:id", a.GetUser)
	put(a, authn, "/api/users/:id", a.UpdateUser)
	delete(a, authn, "/api/users/:id", a.DeleteUser)
	get(a, authn, "/api/users/:id/groups", a.ListUserGroups)

	get(a, authn, "/api/access-keys", a.ListAccessKeys)
	post(a, authn, "/api/access-keys", a.CreateAccessKey)
	delete(a, authn, "/api/access-keys/:id", a.DeleteAccessKey)

	get(a, authn, "/api/groups", a.ListGroups)
	post(a, authn, "/api/groups", a.CreateGroup)
	get(a, authn, "/api/groups/:id", a.GetGroup)

	get(a, authn, "/api/grants", a.ListGrants)
	get(a, authn, "/api/grants/:id", a.GetGrant)
	post(a, authn, "/api/grants", a.CreateGrant)
	delete(a, authn, "/api/grants/:id", a.DeleteGrant)

	post(a, authn, "/api/providers", a.CreateProvider)
	put(a, authn, "/api/providers/:id", a.UpdateProvider)
	delete(a, authn, "/api/providers/:id", a.DeleteProvider)

	get(a, authn, "/api/destinations", a.ListDestinations)
	get(a, authn, "/api/destinations/:id", a.GetDestination)
	post(a, authn, "/api/destinations", a.CreateDestination)
	put(a, authn, "/api/destinations/:id", a.UpdateDestination)
	delete(a, authn, "/api/destinations/:id", a.DeleteDestination)

	post(a, authn, "/api/tokens", a.CreateToken)
	post(a, authn, "/api/logout", a.Logout)

	authn.GET("/api/debug/pprof/*profile", a.pprofHandler)

	// these endpoints do not require authentication
	noAuthn := api.Group("/")
	get(a, noAuthn, "/api/signup", a.SignupEnabled)
	post(a, noAuthn, "/api/signup", a.Signup)

	post(a, noAuthn, "/api/login", a.Login)

	get(a, noAuthn, "/api/providers", a.ListProviders)
	get(a, noAuthn, "/api/providers/:id", a.GetProvider)

	get(a, noAuthn, "/api/version", a.Version)

	// Deprecated in 0.12
	// TODO: remove after a couple versions
	get(a, authn, "/v1/users/:id/grants", a.ListUserGrants)
	get(a, authn, "/v1/groups/:id/grants", a.ListGroupGrants)

	noAuthn.GET("/v1/machines", removed("v0.9.0"))
	noAuthn.POST("/v1/machines", removed("v0.9.0"))
	noAuthn.GET("/v1/machines/:id", removed("v0.9.0"))
	noAuthn.DELETE("/v1/machines/:id", removed("v0.9.0"))
	noAuthn.GET("/v1/machines/:id/grants", removed("v0.9.0"))
	noAuthn.GET("/v1/setup", removed("v0.11.0"))
	noAuthn.GET("/v1/introspect", removed("v0.12.0"))

	// registerUIRoutes must happen last because it uses catch-all middleware
	// with no handlers. Any route added after the UI will end up using the
	// UI middleware unnecessarily.
	// This is a limitation because we serve the UI from / instead of a specific
	// path prefix.
	registerUIRoutes(router, s.options.UI)
	return Routes{Handler: router, OpenAPIDocument: a.openAPIDoc}
}

type ReqHandlerFunc[Req any] func(c *gin.Context, req *Req) error
type ResHandlerFunc[Res any] func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

func get[Req, Res any](a *API, r *gin.RouterGroup, route string, handler ReqResHandlerFunc[Req, Res]) {
	fullPath := path.Join(r.BasePath(), route)
	register(a, http.MethodGet, fullPath, handler)

	wrappedHandler := func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		c.JSON(http.StatusOK, resp)
	}
	bindRoute(a, http.MethodGet, fullPath, wrappedHandler, r.GET)
}

func post[Req, Res any](a *API, r *gin.RouterGroup, route string, handler ReqResHandlerFunc[Req, Res]) {
	fullPath := path.Join(r.BasePath(), route)
	register(a, http.MethodPost, fullPath, handler)

	wh := func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		a.t.RouteEvent(c, fullPath, Properties{"method": "post"})

		c.JSON(http.StatusCreated, resp)
	}
	bindRoute(a, http.MethodPost, fullPath, wh, r.POST)
}

func put[Req, Res any](a *API, r *gin.RouterGroup, route string, handler ReqResHandlerFunc[Req, Res]) {
	fullPath := path.Join(r.BasePath(), route)
	register(a, http.MethodPut, fullPath, handler)

	wh := func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		a.t.RouteEvent(c, fullPath, Properties{"method": "put"})

		c.JSON(http.StatusOK, resp)
	}
	bindRoute(a, http.MethodPut, fullPath, wh, r.PUT)
}

func delete[Req any](a *API, r *gin.RouterGroup, route string, handler ReqHandlerFunc[Req]) {
	fullPath := path.Join(r.BasePath(), route)
	registerDelete(a, http.MethodDelete, fullPath, handler)

	wh := func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, err)
			return
		}

		err := handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		a.t.RouteEvent(c, fullPath, Properties{"method": "delete"})

		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
	}
	bindRoute(a, http.MethodDelete, fullPath, wh, r.DELETE)
}

func bind(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindUri(req); err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	if err := c.ShouldBindQuery(req); err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(req); err != nil {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
	}

	if err := validate.Struct(req); err != nil {
		return err
	}

	return nil
}

func init() {
	gin.DisableBindValidation()
}

type WellKnownJWKResponse struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

func (a *API) wellKnownJWKsHandler(c *gin.Context) {
	keys, err := access.GetPublicJWK(c)
	if err != nil {
		sendAPIError(c, err)
		return
	}

	c.JSON(http.StatusOK, WellKnownJWKResponse{
		Keys: keys,
	})
}

func healthHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

// TODO: use the HTTP Accept header instead of the path to determine the
// format of the response body. https://github.com/infrahq/infra/issues/1610
func (a *API) notFoundHandler(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		sendAPIError(c, internal.ErrNotFound)
		return
	}

	c.Status(http.StatusNotFound)
	buf, err := assetFS.ReadFile("ui/404.html")
	if err != nil {
		logging.S.Error(err)
	}

	_, err = c.Writer.Write(buf)
	if err != nil {
		logging.S.Error(err)
	}
}
