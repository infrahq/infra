package server

import (
	"fmt"
	"net/http"
	"strings"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/metrics"
)

// GenerateRoutes constructs a http.Handler for the primary http and https servers.
// The handler includes gin middleware, API routes, UI routes, and others.
//
// As a side effect of building the handler all API endpoints are registered in
// the global openAPISchema.
//
// The order of routes in this function is important! Gin saves a route along
// with all the middleware that will apply to the route when the
// Router.{GET,POST,etc} method is called.
func (s *Server) GenerateRoutes(promRegistry prometheus.Registerer) (*gin.Engine, error) {
	a := &API{t: s.tel, server: s}
	router := gin.New()
	router.NoRoute(a.notFoundHandler)

	router.Use(gin.Recovery())
	router.GET("/healthz", healthHandler)

	// This group of middleware will apply to everything, including the UI
	router.Use(
		logging.Middleware(),
		RequestTimeoutMiddleware(),
	)

	// This group of middleware only applies to non-ui routes
	api := router.Group("/",
		sentrygin.New(sentrygin.Options{}),
		metrics.Middleware(promRegistry),
		DatabaseMiddleware(a.server.db),
	)
	api.GET("/.well-known/jwks.json", a.wellKnownJWKsHandler)

	authn := api.Group("/", AuthenticationMiddleware(a))
	get(a, authn, "/v1/identities", a.ListIdentities)
	post(a, authn, "/v1/identities", a.CreateIdentity)
	get(a, authn, "/v1/identities/:id", a.GetIdentity)
	put(a, authn, "/v1/identities/:id", a.UpdateIdentity)
	delete(a, authn, "/v1/identities/:id", a.DeleteIdentity)
	get(a, authn, "/v1/identities/:id/groups", a.ListIdentityGroups)
	get(a, authn, "/v1/identities/:id/grants", a.ListIdentityGrants)

	get(a, authn, "/v1/access-keys", a.ListAccessKeys)
	post(a, authn, "/v1/access-keys", a.CreateAccessKey)
	delete(a, authn, "/v1/access-keys/:id", a.DeleteAccessKey)

	get(a, authn, "/v1/groups", a.ListGroups)
	post(a, authn, "/v1/groups", a.CreateGroup)
	get(a, authn, "/v1/groups/:id", a.GetGroup)
	get(a, authn, "/v1/groups/:id/grants", a.ListGroupGrants)

	get(a, authn, "/v1/grants", a.ListGrants)
	get(a, authn, "/v1/grants/:id", a.GetGrant)
	post(a, authn, "/v1/grants", a.CreateGrant)
	delete(a, authn, "/v1/grants/:id", a.DeleteGrant)

	post(a, authn, "/v1/providers", a.CreateProvider)
	put(a, authn, "/v1/providers/:id", a.UpdateProvider)
	delete(a, authn, "/v1/providers/:id", a.DeleteProvider)

	get(a, authn, "/v1/destinations", a.ListDestinations)
	get(a, authn, "/v1/destinations/:id", a.GetDestination)
	post(a, authn, "/v1/destinations", a.CreateDestination)
	put(a, authn, "/v1/destinations/:id", a.UpdateDestination)
	delete(a, authn, "/v1/destinations/:id", a.DeleteDestination)

	get(a, authn, "/v1/introspect", a.Introspect)
	post(a, authn, "/v1/tokens", a.CreateToken)
	post(a, authn, "/v1/logout", a.Logout)

	authn.GET("/v1/debug/pprof/*profile", a.pprofHandler)

	// these endpoints do not require authentication
	noAuthn := api.Group("/")
	get(a, noAuthn, "/v1/setup", a.SetupRequired)
	post(a, noAuthn, "/v1/setup", a.Setup)

	post(a, noAuthn, "/v1/login", a.Login)

	get(a, noAuthn, "/v1/providers", a.ListProviders)
	get(a, noAuthn, "/v1/providers/:id", a.GetProvider)

	get(a, noAuthn, "/v1/version", a.Version)

	// TODO: remove after a couple version.
	noAuthn.GET("/v1/users", removed("v0.9.0"))
	noAuthn.POST("/v1/users", removed("v0.9.0"))
	noAuthn.GET("/v1/users/:id", removed("v0.9.0"))
	noAuthn.PUT("/v1/users/:id", removed("v0.9.0"))
	noAuthn.DELETE("/v1/users/:id", removed("v0.9.0"))
	noAuthn.GET("/v1/users/:id/groups", removed("v0.9.0"))
	noAuthn.GET("/v1/users/:id/grants", removed("v0.9.0"))
	noAuthn.GET("/v1/machines", removed("v0.9.0"))
	noAuthn.POST("/v1/machines", removed("v0.9.0"))
	noAuthn.GET("/v1/machines/:id", removed("v0.9.0"))
	noAuthn.DELETE("/v1/machines/:id", removed("v0.9.0"))
	noAuthn.GET("/v1/machines/:id/grants", removed("v0.9.0"))

	// registerUIRoutes must happen last because it uses catch-all middleware
	// with no handlers. Any route added after the UI will end up using the
	// UI middleware unnecessarily.
	// This is a limitation because we serve the UI from / instead of a specific
	// path prefix.
	if err := registerUIRoutes(router, s.options.UI); err != nil {
		return nil, err
	}
	return router, nil
}

type ReqHandlerFunc[Req any] func(c *gin.Context, req *Req) error
type ResHandlerFunc[Res any] func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

func get[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("GET", r.BasePath(), path, handler)
	fullPathStr := fullPath(r, path)
	r.GET(path, func(c *gin.Context) {
		c.Set("path", fullPathStr)
		req := new(Req)
		if err := bind(c, req); err != nil {
			a.sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			a.sendAPIError(c, err)
			return
		}

		a.t.Event(c, fullPathStr, Properties{"method": "get"})

		c.JSON(http.StatusOK, resp)
	})
}

func post[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("POST", r.BasePath(), path, handler)
	fullPathStr := fullPath(r, path)

	r.POST(path, func(c *gin.Context) {
		c.Set("path", fullPathStr)
		req := new(Req)
		if err := bind(c, req); err != nil {
			a.sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			a.sendAPIError(c, err)
			return
		}

		a.t.Event(c, fullPathStr, Properties{"method": "post"})

		c.JSON(http.StatusCreated, resp)
	})
}

func put[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("PUT", r.BasePath(), path, handler)

	fullPathStr := fullPath(r, path)

	r.PUT(path, func(c *gin.Context) {
		c.Set("path", fullPathStr)
		req := new(Req)
		if err := bind(c, req); err != nil {
			a.sendAPIError(c, err)
			return
		}

		resp, err := handler(c, req)
		if err != nil {
			a.sendAPIError(c, err)
			return
		}

		a.t.Event(c, fullPathStr, Properties{"method": "put"})

		c.JSON(http.StatusOK, resp)
	})
}

func delete[Req any](a *API, r *gin.RouterGroup, path string, handler ReqHandlerFunc[Req]) {
	registerReq("DELETE", r.BasePath(), path, handler)

	fullPathStr := fullPath(r, path)

	r.DELETE(path, func(c *gin.Context) {
		c.Set("path", fullPathStr)
		req := new(Req)
		if err := bind(c, req); err != nil {
			a.sendAPIError(c, err)
			return
		}

		err := handler(c, req)
		if err != nil {
			a.sendAPIError(c, err)
			return
		}

		a.t.Event(c, fullPathStr, Properties{"method": "delete"})

		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
	})
}

func fullPath(r *gin.RouterGroup, path string) string {
	return strings.TrimRight(r.BasePath(), "/") + "/" + strings.TrimLeft(path, "/")
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
		a.sendAPIError(c, err)
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
	if strings.HasPrefix(c.Request.URL.Path, "/v1") {
		a.sendAPIError(c, internal.ErrNotFound)
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
