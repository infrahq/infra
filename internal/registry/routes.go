package registry

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

type ReqHandlerFunc[Req any] func(c *gin.Context, req *Req) error
type ResHandlerFunc[Res any] func(c *gin.Context) (Res, error)
type ReqResHandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

func (a *API) registerRoutes(router *gin.RouterGroup) {
	router.Use(
		RequestTimeoutMiddleware(),
		DatabaseMiddleware(a.registry.db),
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

func get[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	r.GET(path, MetricsMiddleware(path), func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
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
	})
}

func post[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	r.POST(path, MetricsMiddleware(path), func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
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
	})
}

func put[Req, Res any](r *gin.RouterGroup, path string, handler ReqResHandlerFunc[Req, Res]) {
	r.PUT(path, MetricsMiddleware(path), func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
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
	})
}

func delete[Req any](r *gin.RouterGroup, path string, handler ReqHandlerFunc[Req]) {
	r.DELETE(path, MetricsMiddleware(path), func(c *gin.Context) {
		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, fmt.Errorf("%w: %s", internal.ErrBadRequest, err))
			return
		}
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
	})
}

func bind(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindUri(req); err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}
	if err := c.ShouldBindQuery(req); err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(req); err != nil {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
	}

	return nil
}

func init() {
	gin.DisableBindValidation()
}
