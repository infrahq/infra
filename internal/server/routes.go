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

type ReqHandlerFunc[Req any]         func(c *gin.Context, req *Req) error
type ResHandlerFunc[Res any]         func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

func (a *API) registerRoutes(router *gin.RouterGroup) {
	router.Use(
		sentrygin.New(sentrygin.Options{}),
		metrics.Middleware(),
		logging.IdentityAwareMiddleware(),
		logging.Middleware(),
		RequestTimeoutMiddleware(),
		DatabaseMiddleware(a.server.db),
	)

	authorized := router.Group("/",
		AuthenticationMiddleware(),
	)

	{
		get(authorized, "/users", a.ListUsers)
		post(authorized, "/users", a.CreateUser)
		get(authorized, "/users/:id", a.GetUser)
		put(authorized, "/users/:id", a.UpdateUser)
		delete(authorized, "/users/:id", a.DeleteUser)
		get(authorized, "/users/:id/groups", a.ListUserGroups)
		get(authorized, "/users/:id/grants", a.ListUserGrants)

		get(authorized, "/machines", a.ListMachines)
		post(authorized, "/machines", a.CreateMachine)
		get(authorized, "/machines/:id", a.GetMachine)
		delete(authorized, "/machines/:id", a.DeleteMachine)
		get(authorized, "/machines/:id/grants", a.ListMachineGrants)

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
	unauthorized := router.Group("/")

	{
		get(unauthorized, "/setup", a.SetupRequired)
		post(unauthorized, "/setup", a.Setup)

		post(unauthorized, "/login", a.Login)

		get(unauthorized, "/providers", a.ListProviders)
		get(unauthorized, "/providers/:id", a.GetProvider)

		get(unauthorized, "/version", a.Version)
	}

	generateOpenAPI()
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
