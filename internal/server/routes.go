package server

import (
	"fmt"
	"net/http"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/square/go-jose.v2"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/metrics"
)

type ReqHandlerFunc[Req any] func(c *gin.Context, req *Req) error
type ResHandlerFunc[Res any] func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

func (a *API) registerRoutes(router *gin.RouterGroup, promRegistry prometheus.Registerer) {
	router.GET("/healthz", a.healthHandler)
	router.GET("/.well-known/jwks.json", DatabaseMiddleware(a.server.db), a.wellKnownJWKsHandler)

	router.Use(
		sentrygin.New(sentrygin.Options{}),
		metrics.Middleware(promRegistry),
		logging.IdentityAwareMiddleware(),
		DatabaseMiddleware(a.server.db),
	)

	v1 := router.Group("/v1")
	authorized := v1.Group("/", AuthenticationMiddleware(a))

	{
		get(a, authorized, "/identities", a.ListIdentities)
		post(a, authorized, "/identities", a.CreateIdentity)
		get(a, authorized, "/identities/:id", a.GetIdentity)
		put(a, authorized, "/identities/:id", a.UpdateIdentity)
		delete(a, authorized, "/identities/:id", a.DeleteIdentity)
		get(a, authorized, "/identities/:id/groups", a.ListIdentityGroups)
		get(a, authorized, "/identities/:id/grants", a.ListIdentityGrants)

		get(a, authorized, "/access-keys", a.ListAccessKeys)
		post(a, authorized, "/access-keys", a.CreateAccessKey)
		delete(a, authorized, "/access-keys/:id", a.DeleteAccessKey)

		get(a, authorized, "/introspect", a.Introspect)

		get(a, authorized, "/groups", a.ListGroups)
		post(a, authorized, "/groups", a.CreateGroup)
		get(a, authorized, "/groups/:id", a.GetGroup)
		get(a, authorized, "/groups/:id/grants", a.ListGroupGrants)

		get(a, authorized, "/grants", a.ListGrants)
		get(a, authorized, "/grants/:id", a.GetGrant)
		post(a, authorized, "/grants", a.CreateGrant)
		delete(a, authorized, "/grants/:id", a.DeleteGrant)

		post(a, authorized, "/providers", a.CreateProvider)
		put(a, authorized, "/providers/:id", a.UpdateProvider)
		delete(a, authorized, "/providers/:id", a.DeleteProvider)

		get(a, authorized, "/destinations", a.ListDestinations)
		get(a, authorized, "/destinations/:id", a.GetDestination)
		post(a, authorized, "/destinations", a.CreateDestination)
		put(a, authorized, "/destinations/:id", a.UpdateDestination)
		delete(a, authorized, "/destinations/:id", a.DeleteDestination)

		post(a, authorized, "/tokens", a.CreateToken)

		post(a, authorized, "/logout", a.Logout)
	}

	// these endpoints are left unauthenticated
	unauthorized := v1.Group("/")

	{
		get(a, unauthorized, "/setup", a.SetupRequired)
		post(a, unauthorized, "/setup", a.Setup)

		post(a, unauthorized, "/login", a.Login)

		get(a, unauthorized, "/providers", a.ListProviders)
		get(a, unauthorized, "/providers/:id", a.GetProvider)

		get(a, unauthorized, "/version", a.Version)
	}

	// pprof.Index does not work with a /v1 prefix
	debug := router.Group("/debug/pprof", AuthenticationMiddleware(a))
	debug.GET("/*profile", a.pprofHandler)

	// TODO: remove after a couple version.
	v1.GET("/users", removed("v0.9.0"))
	v1.POST("/users", removed("v0.9.0"))
	v1.GET("/users/:id", removed("v0.9.0"))
	v1.PUT("/users/:id", removed("v0.9.0"))
	v1.DELETE("/users/:id", removed("v0.9.0"))
	v1.GET("/users/:id/groups", removed("v0.9.0"))
	v1.GET("/users/:id/grants", removed("v0.9.0"))
	v1.GET("/machines", removed("v0.9.0"))
	v1.POST("/machines", removed("v0.9.0"))
	v1.GET("/machines/:id", removed("v0.9.0"))
	v1.DELETE("/machines/:id", removed("v0.9.0"))
	v1.GET("/machines/:id/grants", removed("v0.9.0"))
}

func get[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("GET", r.BasePath(), path, handler)
	r.GET(path, func(c *gin.Context) {
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

		c.JSON(http.StatusOK, resp)
	})
}

func post[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("POST", r.BasePath(), path, handler)
	r.POST(path, func(c *gin.Context) {
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

		c.JSON(http.StatusCreated, resp)
	})
}

func put[Req, Res any](a *API, r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("PUT", r.BasePath(), path, handler)
	r.PUT(path, func(c *gin.Context) {
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

		c.JSON(http.StatusOK, resp)
	})
}

func delete[Req any](a *API, r *gin.RouterGroup, path string, handler ReqHandlerFunc[Req]) {
	registerReq("DELETE", r.BasePath(), path, handler)
	r.DELETE(path, func(c *gin.Context) {
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

		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
	})
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

func (a *API) healthHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}
