package server

import (
	"fmt"
	"net/http"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/metrics"
)

type (
	ReqHandlerFunc[Req any]         func(c *gin.Context, req *Req) error
	ResHandlerFunc[Res any]         func(c *gin.Context) (Res, error)
	ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)
)

func (a *API) registerRoutes(router *gin.Engine) {
	router.Use(
		sentrygin.New(sentrygin.Options{}),
		metrics.Middleware(),
		logging.IdentityAwareMiddleware(),
		logging.Middleware(),
		RequestTimeoutMiddleware(),
		DatabaseMiddleware(a.server.db),
	)

	v1 := router.Group("/v1")
	authorized := v1.Group("/", AuthenticationMiddleware())

	{
		get(authorized, "/identities", a.ListIdentities)
		post(authorized, "/identities", a.CreateIdentity)
		get(authorized, "/identities/:id", a.GetIdentity)
		put(authorized, "/identities/:id", a.UpdateIdentity)
		delete(authorized, "/identities/:id", a.DeleteIdentity)
		get(authorized, "/identities/:id/groups", a.ListIdentityGroups)
		get(authorized, "/identities/:id/grants", a.ListIdentityGrants)

		get(authorized, "/access-keys", a.ListAccessKeys)
		post(authorized, "/access-keys", a.CreateAccessKey)
		delete(authorized, "/access-keys/:id", a.DeleteAccessKey)

		get(authorized, "/introspect", a.Introspect)

		get(authorized, "/groups", a.ListGroups)
		post(authorized, "/groups", a.CreateGroup)
		get(authorized, "/groups/:id", a.GetGroup)
		get(authorized, "/groups/:id/grants", a.ListGroupGrants)

		get(authorized, "/grants", a.ListGrants)
		get(authorized, "/grants/:id", a.GetGrant)
		post(authorized, "/grants", a.CreateGrant)
		delete(authorized, "/grants/:id", a.DeleteGrant)

		post(authorized, "/providers", a.CreateProvider)
		put(authorized, "/providers/:id", a.UpdateProvider)
		delete(authorized, "/providers/:id", a.DeleteProvider)

		get(authorized, "/destinations", a.ListDestinations)
		get(authorized, "/destinations/:id", a.GetDestination)
		post(authorized, "/destinations", a.CreateDestination)
		put(authorized, "/destinations/:id", a.UpdateDestination)
		delete(authorized, "/destinations/:id", a.DeleteDestination)

		post(authorized, "/tokens", a.CreateToken)

		post(authorized, "/logout", a.Logout)
	}

	// these endpoints are left unauthenticated
	unauthorized := v1.Group("/")

	{
		get(unauthorized, "/setup", a.SetupRequired)
		post(unauthorized, "/setup", a.Setup)

		post(unauthorized, "/login", a.Login)

		get(unauthorized, "/providers", a.ListProviders)
		get(unauthorized, "/providers/:id", a.GetProvider)

		get(unauthorized, "/version", a.Version)
	}

	// pprof.Index does not work with a /v1 prefix
	debug := router.Group("/debug/pprof", AuthenticationMiddleware())
	debug.GET("/*profile", pprofHandler)

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

func get[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("GET", r.BasePath(), path, handler)
	r.GET(path, func(c *gin.Context) {
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
	})
}

func post[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("POST", r.BasePath(), path, handler)
	r.POST(path, func(c *gin.Context) {
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

		c.JSON(http.StatusCreated, resp)
	})
}

func put[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	register("PUT", r.BasePath(), path, handler)
	r.PUT(path, func(c *gin.Context) {
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
	})
}

func delete[Req any](r *gin.RouterGroup, path string, handler ReqHandlerFunc[Req]) {
	registerReq("DELETE", r.BasePath(), path, handler)
	r.DELETE(path, func(c *gin.Context) {
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
