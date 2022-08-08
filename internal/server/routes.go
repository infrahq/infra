package server

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/validate"
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
	a.addRewrites()
	a.addRedirects()

	router := gin.New()
	router.NoRoute(a.notFoundHandler)

	router.Use(gin.Recovery())
	router.GET("/healthz", healthHandler)

	// This group of middleware will apply to everything, including the UI
	router.Use(
		logging.Middleware(),
		TimeoutMiddleware(1*time.Minute),
	)

	// This group of middleware only applies to non-ui routes
	apiGroup := router.Group("/", metrics.Middleware(promRegistry))

	authn := apiGroup.Group("/", authenticatedMiddleware(a.server))

	get(a, authn, "/api/users", a.ListUsers)
	post(a, authn, "/api/users", a.CreateUser)
	get(a, authn, "/api/users/:id", a.GetUser)
	put(a, authn, "/api/users/:id", a.UpdateUser)
	del(a, authn, "/api/users/:id", a.DeleteUser)

	get(a, authn, "/api/access-keys", a.ListAccessKeys)
	post(a, authn, "/api/access-keys", a.CreateAccessKey)
	del(a, authn, "/api/access-keys/:id", a.DeleteAccessKey)

	get(a, authn, "/api/groups", a.ListGroups)
	post(a, authn, "/api/groups", a.CreateGroup)
	get(a, authn, "/api/groups/:id", a.GetGroup)
	del(a, authn, "/api/groups/:id", a.DeleteGroup)
	patch(a, authn, "/api/groups/:id/users", a.UpdateUsersInGroup)

	get(a, authn, "/api/organizations", a.ListOrganizations)
	post(a, authn, "/api/organizations", a.CreateOrganization)
	get(a, authn, "/api/organizations/:id", a.GetOrganization)
	del(a, authn, "/api/organizations/:id", a.DeleteOrganization)

	get(a, authn, "/api/grants", a.ListGrants)
	get(a, authn, "/api/grants/:id", a.GetGrant)
	post(a, authn, "/api/grants", a.CreateGrant)
	del(a, authn, "/api/grants/:id", a.DeleteGrant)

	post(a, authn, "/api/providers", a.CreateProvider)
	put(a, authn, "/api/providers/:id", a.UpdateProvider)
	del(a, authn, "/api/providers/:id", a.DeleteProvider)

	get(a, authn, "/api/destinations", a.ListDestinations)
	get(a, authn, "/api/destinations/:id", a.GetDestination)
	post(a, authn, "/api/destinations", a.CreateDestination)
	put(a, authn, "/api/destinations/:id", a.UpdateDestination)
	del(a, authn, "/api/destinations/:id", a.DeleteDestination)

	post(a, authn, "/api/tokens", a.CreateToken)
	post(a, authn, "/api/logout", a.Logout)

	authn.GET("/api/debug/pprof/*profile", pprofHandler)

	// these endpoints do not require authentication
	noAuthn := apiGroup.Group("/", unauthenticatedMiddleware(a.server))
	post(a, noAuthn, "/api/signup", a.Signup)

	post(a, noAuthn, "/api/login", a.Login)
	post(a, noAuthn, "/api/password-reset-request", a.RequestPasswordReset)
	post(a, noAuthn, "/api/password-reset", a.VerifiedPasswordReset)

	get(a, noAuthn, "/api/providers", a.ListProviders)
	get(a, noAuthn, "/api/providers/:id", a.GetProvider)

	get(a, noAuthn, "/api/version", a.Version)

	// deprecated endpoints
	// CLI clients before v0.14.4 rely on sign-up being false to continue with login
	type SignupEnabledResponse struct {
		Enabled bool `json:"enabled"`
	}
	addDeprecated(a, noAuthn, http.MethodGet, "/api/signup",
		func(c *gin.Context, _ *api.EmptyRequest) (*SignupEnabledResponse, error) {
			return &SignupEnabledResponse{Enabled: false}, nil
		},
	)

	get(a, noAuthn, "/api/settings", a.GetSettings)
	put(a, authn, "/api/settings", a.UpdateSettings)
	add(a, noAuthn, route[api.EmptyRequest, WellKnownJWKResponse]{
		method:              http.MethodGet,
		path:                "/.well-known/jwks.json",
		handler:             wellKnownJWKsHandler,
		omitFromDocs:        true,
		omitFromTelemetry:   true,
		infraHeaderOptional: true,
	})

	// registerUIRoutes must happen last because it uses catch-all middleware
	// with no handlers. Any route added after the UI will end up using the
	// UI middleware unnecessarily.
	// This is a limitation because we serve the UI from / instead of a specific
	// path prefix.
	registerUIRoutes(router, s.options.UI)
	return Routes{Handler: router, OpenAPIDocument: a.openAPIDoc}
}

type HandlerFunc[Req, Res any] func(c *gin.Context, req *Req) (Res, error)

type route[Req, Res any] struct {
	method              string
	path                string
	handler             HandlerFunc[Req, Res]
	omitFromDocs        bool
	omitFromTelemetry   bool
	infraHeaderOptional bool
}

func add[Req, Res any](a *API, r *gin.RouterGroup, route route[Req, Res]) {
	route.path = path.Join(r.BasePath(), route.path)

	if !route.omitFromDocs {
		a.register(openAPIRouteDefinition(route))
	}

	wrappedHandler := func(c *gin.Context) {
		if !route.infraHeaderOptional {
			if _, err := requestVersion(c.Request); err != nil {
				sendAPIError(c, err)
				return
			}
		}

		req := new(Req)
		if err := bind(c, req); err != nil {
			sendAPIError(c, err)
			return
		}

		trimWhitespace(req)

		resp, err := route.handler(c, req)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		if !route.omitFromTelemetry {
			a.t.RouteEvent(c, route.path, Properties{"method": strings.ToLower(route.method)})
		}

		statusCode := defaultResponseCodeForMethod(route.method)
		if c, ok := any(resp).(statusCoder); ok {
			if code := c.StatusCode(); code != 0 {
				statusCode = code
			}
		}

		c.JSON(statusCode, resp)
	}

	bindRoute(a, r, route.method, route.path, wrappedHandler)
}

type statusCoder interface {
	StatusCode() int
}

var reflectTypeString = reflect.TypeOf("")

// trimWhitespace trims leading and trailing whitespace from any string fields
// in req. The req argument must be a non-nil pointer to a struct.
func trimWhitespace(req interface{}) {
	v := reflect.Indirect(reflect.ValueOf(req))
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Type() == reflectTypeString {
				f.SetString(strings.TrimSpace(f.String()))
			}
		}
	}
}

func defaultResponseCodeForMethod(method string) int {
	switch method {
	case http.MethodPost:
		return http.StatusCreated
	case http.MethodDelete:
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}

func get[Req, Res any](a *API, r *gin.RouterGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{
		method:            http.MethodGet,
		path:              path,
		handler:           handler,
		omitFromTelemetry: true,
	})
}

func post[Req, Res any](a *API, r *gin.RouterGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{method: http.MethodPost, path: path, handler: handler})
}

func put[Req, Res any](a *API, r *gin.RouterGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{method: http.MethodPut, path: path, handler: handler})
}

func patch[Req, Res any](a *API, r *gin.RouterGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{method: http.MethodPatch, path: path, handler: handler})
}

func del[Req any, Res any](a *API, r *gin.RouterGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{method: http.MethodDelete, path: path, handler: handler})
}

func addDeprecated[Req, Res any](a *API, r *gin.RouterGroup, method string, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, route[Req, Res]{
		method:            method,
		path:              path,
		handler:           handler,
		omitFromTelemetry: true,
		omitFromDocs:      true,
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

	if r, ok := req.(validate.Request); ok {
		if err := validate.Validate(r); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	gin.DisableBindValidation()
}

func healthHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (a *API) notFoundHandler(c *gin.Context) {
	accept := c.Request.Header.Get("Accept")
	if strings.HasPrefix(accept, "application/json") {
		sendAPIError(c, internal.ErrNotFound)
		return
	}

	c.Status(http.StatusNotFound)

	_, err := c.Writer.Write([]byte("404 not found"))
	if err != nil {
		logging.Errorf("%s", err.Error())
	}
}
