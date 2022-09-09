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
func (s *Server) GenerateRoutes() Routes {
	a := &API{t: s.tel, server: s}
	a.addRewrites()
	a.addRedirects()

	router := gin.New()
	router.NoRoute(a.notFoundHandler)

	router.Use(gin.Recovery())
	router.GET("/healthz", healthHandler)

	// This group of middleware will apply to everything, including the UI
	router.Use(
		loggingMiddleware(s.options.EnableLogSampling),
		TimeoutMiddleware(1*time.Minute),
	)

	// This group of middleware only applies to non-ui routes
	apiGroup := router.Group("/", metrics.Middleware(s.metricsRegistry))

	// auth required, org required
	authn := &routeGroup{RouterGroup: apiGroup.Group("/")}

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

	put(a, authn, "/api/settings", a.UpdateSettings)

	add(a, authn, http.MethodGet, "/api/debug/pprof/*profile", pprofRoute)

	// no auth required, org not required
	noAuthnNoOrg := &routeGroup{RouterGroup: apiGroup.Group("/"), noAuthentication: true, noOrgRequired: true}
	post(a, noAuthnNoOrg, "/api/signup", a.Signup)
	get(a, noAuthnNoOrg, "/api/version", a.Version)
	get(a, noAuthnNoOrg, "/api/server-configuration", a.GetServerConfiguration)
	post(a, noAuthnNoOrg, "/api/forgot-domain-request", a.RequestForgotDomains)

	// no auth required, org required
	noAuthnWithOrg := &routeGroup{RouterGroup: apiGroup.Group("/"), noAuthentication: true}

	post(a, noAuthnWithOrg, "/api/login", a.Login)
	post(a, noAuthnWithOrg, "/api/password-reset-request", a.RequestPasswordReset)
	post(a, noAuthnWithOrg, "/api/password-reset", a.VerifiedPasswordReset)

	get(a, noAuthnWithOrg, "/api/providers/:id", a.GetProvider)
	get(a, noAuthnWithOrg, "/api/providers", a.ListProviders)
	get(a, noAuthnWithOrg, "/api/settings", a.GetSettings)

	add(a, noAuthnWithOrg, http.MethodGet, "/.well-known/jwks.json", wellKnownJWKsRoute)

	a.deprecatedRoutes(noAuthnNoOrg)

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
	handler                    HandlerFunc[Req, Res]
	omitFromDocs               bool
	omitFromTelemetry          bool
	infraVersionHeaderOptional bool
	noAuthentication           bool
	noOrgRequired              bool
}

type routeIdentifier struct {
	method string
	path   string
}

// TODO: replace this when routes are defined as package-level vars instead of
// constructed from the get, post, put, del helper functions.
type routeGroup struct {
	*gin.RouterGroup
	noAuthentication bool
	noOrgRequired    bool
}

func add[Req, Res any](a *API, group *routeGroup, method, urlPath string, route route[Req, Res]) {
	routeID := routeIdentifier{
		method: method,
		path:   path.Join(group.BasePath(), urlPath),
	}

	if !route.omitFromDocs {
		a.register(openAPIRouteDefinition(routeID, route))
	}

	route.noAuthentication = group.noAuthentication
	route.noOrgRequired = group.noOrgRequired

	handler := func(c *gin.Context) {
		if err := wrapRoute(a, routeID, route)(c); err != nil {
			sendAPIError(c, err)
		}
	}
	bindRoute(a, group.RouterGroup, routeID, handler)
}

// wrapRoute builds a gin.HandlerFunc from a route. The returned function
// provides functionality that is applicable to a large number of routes
// (similar to middleware).
// The returned function handles validation of the infra version header, manages
// a request scoped database transaction, authenticates the request, reads the
// request fields into a request struct, and returns an HTTP response with a
// status code and response body built from the response type.
func wrapRoute[Req, Res any](a *API, routeID routeIdentifier, route route[Req, Res]) func(*gin.Context) error {
	return func(c *gin.Context) error {
		if !route.infraVersionHeaderOptional {
			if _, err := requestVersion(c.Request); err != nil {
				return err
			}
		}

		tx, err := a.server.db.Begin(c.Request.Context())
		if err != nil {
			return err
		}
		defer func() {
			if err := tx.Rollback(); err != nil {
				logging.L.Error().Err(err).Msg("failed to rollback database transaction")
			}
		}()

		if route.noAuthentication {
			err = validateRequestOrganization(c, tx, a.server)
		} else {
			err = authenticateRequest(c, tx, a.server)
		}
		if err != nil {
			return err
		}

		if !route.noOrgRequired {
			if org := getRequestContext(c).Authenticated.Organization; org == nil {
				return internal.ErrBadRequest
			}
		}

		req := new(Req)
		if err := readRequest(c, req); err != nil {
			return err
		}

		resp, err := route.handler(c, req)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		if !route.omitFromTelemetry {
			a.t.RouteEvent(c, routeID.path, Properties{"method": strings.ToLower(routeID.method)})
		}

		c.JSON(responseStatusCode(routeID.method, resp), resp)
		return nil
	}
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

func responseStatusCode(method string, resp any) int {
	if c, ok := resp.(statusCoder); ok {
		if code := c.StatusCode(); code != 0 {
			return code
		}
	}
	switch method {
	case http.MethodPost:
		return http.StatusCreated
	case http.MethodDelete:
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}

func get[Req, Res any](a *API, r *routeGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, http.MethodGet, path, route[Req, Res]{
		handler:           handler,
		omitFromTelemetry: true,
	})
}

func post[Req, Res any](a *API, r *routeGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, http.MethodPost, path, route[Req, Res]{handler: handler})
}

func put[Req, Res any](a *API, r *routeGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, http.MethodPut, path, route[Req, Res]{handler: handler})
}

func patch[Req, Res any](a *API, r *routeGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, http.MethodPatch, path, route[Req, Res]{handler: handler})
}

func del[Req any, Res any](a *API, r *routeGroup, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, http.MethodDelete, path, route[Req, Res]{handler: handler})
}

func addDeprecated[Req, Res any](a *API, r *routeGroup, method string, path string, handler HandlerFunc[Req, Res]) {
	add(a, r, method, path, route[Req, Res]{
		handler:           handler,
		omitFromTelemetry: true,
		omitFromDocs:      true,
	})
}

func readRequest(c *gin.Context, req interface{}) error {
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

	trimWhitespace(req)
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
