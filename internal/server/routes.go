package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/metrics"
)

// Routes is the return value of GenerateRoutes.
type Routes struct {
	http.Handler
	OpenAPIDocument openapi3.T
	api             *API
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

	router := gin.New()
	router.NoRoute(a.notFoundHandler)
	router.GET("/healthz", healthHandler)

	// This group of middleware will apply to everything, including the UI
	router.Use(loggingMiddleware(s.options.EnableLogSampling))

	// This group of middleware only applies to non-ui routes
	apiGroup := router.Group("/", metrics.Middleware(s.metricsRegistry))

	// auth required, org required
	authn := &routeGroup{RouterGroup: apiGroup.Group("/")}

	get(a, authn, "/api/users", a.ListUsers)
	post(a, authn, "/api/users", a.CreateUser)
	add(a, authn, http.MethodGet, "/api/users/:id", getUserRoute)
	put(a, authn, "/api/users/:id", a.UpdateUser)
	del(a, authn, "/api/users/:id", a.DeleteUser)
	put(a, authn, "/api/users/public-key", AddUserPublicKey)

	get(a, authn, "/api/access-keys", a.ListAccessKeys)
	post(a, authn, "/api/access-keys", a.CreateAccessKey)
	del(a, authn, "/api/access-keys/:id", a.DeleteAccessKey)
	del(a, authn, "/api/access-keys", a.DeleteAccessKeys)

	get(a, authn, "/api/groups", a.ListGroups)
	post(a, authn, "/api/groups", a.CreateGroup)
	get(a, authn, "/api/groups/:id", a.GetGroup)
	del(a, authn, "/api/groups/:id", a.DeleteGroup)
	patch(a, authn, "/api/groups/:id/users", a.UpdateUsersInGroup)

	get(a, authn, "/api/organizations", a.ListOrganizations)
	post(a, authn, "/api/organizations", a.CreateOrganization)
	get(a, authn, "/api/organizations/:id", a.GetOrganization)
	del(a, authn, "/api/organizations/:id", a.DeleteOrganization)
	put(a, authn, "/api/organizations/:id", a.UpdateOrganization)

	get(a, authn, "/api/grants", a.ListGrants)
	get(a, authn, "/api/grants/:id", a.GetGrant)
	post(a, authn, "/api/grants", a.CreateGrant)
	del(a, authn, "/api/grants/:id", a.DeleteGrant)
	patch(a, authn, "/api/grants", a.UpdateGrants)

	post(a, authn, "/api/providers", a.CreateProvider)
	patch(a, authn, "/api/providers/:id", a.PatchProvider)
	put(a, authn, "/api/providers/:id", a.UpdateProvider)
	del(a, authn, "/api/providers/:id", a.DeleteProvider)

	get(a, authn, "/api/destinations", a.ListDestinations)
	get(a, authn, "/api/destinations/:id", a.GetDestination)
	post(a, authn, "/api/destinations", a.CreateDestination)
	put(a, authn, "/api/destinations/:id", a.UpdateDestination)
	del(a, authn, "/api/destinations/:id", a.DeleteDestination)

	add(a, authn, http.MethodPost, "/api/tokens", createTokenRoute)
	post(a, authn, "/api/logout", a.Logout)

	// SCIM inbound provisioning
	add(a, authn, http.MethodGet, "/api/scim/v2/Users/:id", getProviderUsersRoute)
	add(a, authn, http.MethodGet, "/api/scim/v2/Users", listProviderUsersRoute)
	add(a, authn, http.MethodPost, "/api/scim/v2/Users", createProviderUserRoute)
	add(a, authn, http.MethodPut, "/api/scim/v2/Users/:id", updateProviderUserRoute)
	add(a, authn, http.MethodPatch, "/api/scim/v2/Users/:id", patchProviderUserRoute)
	add(a, authn, http.MethodDelete, "/api/scim/v2/Users/:id", deleteProviderUserRoute)

	add(a, authn, http.MethodGet, "/api/debug/pprof/*profile", pprofRoute)

	// no auth required, org not required
	noAuthnNoOrg := &routeGroup{RouterGroup: apiGroup.Group("/"), authenticationOptional: true, organizationOptional: true}
	add(a, noAuthnNoOrg, http.MethodPost, "/api/signup", a.SignupRoute())
	get(a, noAuthnNoOrg, "/api/version", a.Version)
	get(a, noAuthnNoOrg, "/api/server-configuration", a.GetServerConfiguration)
	post(a, noAuthnNoOrg, "/api/forgot-domain-request", a.RequestForgotDomains)

	// no auth required, org required
	noAuthnWithOrg := &routeGroup{RouterGroup: apiGroup.Group("/"), authenticationOptional: true}

	post(a, noAuthnWithOrg, "/api/login", a.Login)
	post(a, noAuthnWithOrg, "/api/password-reset-request", a.RequestPasswordReset)
	post(a, noAuthnWithOrg, "/api/password-reset", a.VerifiedPasswordReset)

	get(a, noAuthnWithOrg, "/api/providers/:id", a.GetProvider)
	get(a, noAuthnWithOrg, "/api/providers", a.ListProviders)
	add(a, noAuthnWithOrg, http.MethodGet, "/link", verifyAndRedirectRoute)

	add(a, noAuthnWithOrg, http.MethodGet, "/.well-known/jwks.json", wellKnownJWKsRoute)

	// Device flow
	post(a, noAuthnNoOrg, "/api/device", a.StartDeviceFlow)
	post(a, noAuthnWithOrg, "/api/device/status", a.GetDeviceFlowStatus)
	post(a, authn, "/api/device/approve", a.ApproveDeviceFlow)

	a.deprecatedRoutes(noAuthnNoOrg)
	a.addPreviousVersionHandlersAccessKey()
	a.addPreviousVersionHandlersSignup()
	a.addPreviousVersionHandlersGrants()

	// registerUIRoutes must happen last because it uses catch-all middleware
	// with no handlers. Any route added after the UI will end up using the
	// UI middleware unnecessarily.
	// This is a limitation because we serve the UI from / instead of a specific
	// path prefix.
	registerUIRoutes(router, s.options.UI)
	return Routes{Handler: router, OpenAPIDocument: a.openAPIDoc, api: a}
}

type HandlerFunc[Req, Res any] func(rCtx access.RequestContext, req *Req) (Res, error)

type route[Req, Res any] struct {
	routeSettings
	handler HandlerFunc[Req, Res]
}

type routeSettings struct {
	omitFromDocs               bool
	omitFromTelemetry          bool
	infraVersionHeaderOptional bool
	authenticationOptional     bool
	organizationOptional       bool
	idpSync                    bool // when true the user session will be syncronized with the identity provider on a timed interval
	txnOptions                 *sql.TxOptions
}

type routeIdentifier struct {
	method string
	path   string
}

// TODO: replace this when routes are defined as package-level vars instead of
// constructed from the get, post, put, del helper functions.
type routeGroup struct {
	*gin.RouterGroup
	authenticationOptional bool
	organizationOptional   bool
}

func add[Req, Res any](a *API, group *routeGroup, method, urlPath string, route route[Req, Res]) {
	routeID := routeIdentifier{
		method: method,
		path:   path.Join(group.BasePath(), urlPath),
	}

	route.authenticationOptional = group.authenticationOptional
	route.organizationOptional = group.organizationOptional

	if !route.omitFromDocs {
		a.register(openAPIRouteDefinition(routeID, route))
	}

	handler := func(c *gin.Context) {
		reqVer, err := requestVersion(c.Request)
		if err != nil && !route.infraVersionHeaderOptional {
			sendAPIError(c.Writer, c.Request, err)
			return
		}

		versions := a.versions[routeID]
		if versionedHandler := handlerForVersion(versions, reqVer); versionedHandler != nil {
			versionedHandler(c)
			return
		}

		if err := wrapRoute(a, routeID, route)(c); err != nil {
			sendAPIError(c.Writer, c.Request, err)
			return
		}
	}
	group.RouterGroup.Handle(routeID.method, routeID.path, handler)
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
		origRequestContext := c.Request.Context()
		ctx, cancel := context.WithTimeout(origRequestContext, a.server.options.API.RequestTimeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		authned, err := authenticateRequest(c, route.routeSettings, a.server)
		if err != nil {
			return err
		}

		req := new(Req)
		if err := readRequest(c, req); err != nil {
			return err
		}

		if r, ok := any(req).(isBlockingRequest); ok && r.IsBlockingRequest() {
			ctx, cancel := context.WithTimeout(
				origRequestContext, a.server.options.API.BlockingRequestTimeout)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
		}

		tx, err := a.server.db.Begin(c.Request.Context(), route.txnOptions)
		if err != nil {
			return err
		}
		defer logError(tx.Rollback, "failed to rollback request handler transaction")

		if org := authned.Organization; org != nil {
			tx = tx.WithOrgID(org.ID)
		}
		rCtx := access.RequestContext{
			Request:       c.Request,
			DBTxn:         tx,
			Authenticated: authned,
			DataDB:        a.server.db,
			Response:      &access.Response{HTTPWriter: c.Writer},
		}
		c.Set(access.RequestContextKey, rCtx)

		resp, err := route.handler(c, req)
		if err != nil {
			return err
		}

		completeTx := tx.Commit
		if route.txnOptions != nil && route.txnOptions.ReadOnly {
			// use rollback to avoid an error when the request handler already completed the txn
			completeTx = tx.Rollback
		}
		if err := completeTx(); err != nil {
			return err
		}

		if !route.omitFromTelemetry {
			a.t.RouteEvent(rCtx, routeID.path, Properties{"method": strings.ToLower(routeID.method)})
		}

		// TODO: extract all response header/status/body writing to another function
		if respHeaders, ok := any(resp).(hasResponseHeaders); ok {
			respHeaders.SetHeaders(rCtx.Response.HTTPWriter.Header())
		}
		if r, ok := any(resp).(isRedirect); ok {
			c.Redirect(http.StatusPermanentRedirect, r.RedirectURL())
		} else {
			c.JSON(responseStatusCode(routeID.method, resp), resp)
		}
		return nil
	}
}

type hasResponseHeaders interface {
	SetHeaders(http.Header)
}

type isRedirect interface {
	RedirectURL() string
}

type statusCoder interface {
	StatusCode() int
}

type isBlockingRequest interface {
	IsBlockingRequest() bool
}

func requestVersion(req *http.Request) (*semver.Version, error) {
	headerVer := req.Header.Get("Infra-Version")
	if headerVer == "" {
		return nil, fmt.Errorf("%w: Infra-Version header is required. The current version is %s", internal.ErrBadRequest, internal.FullVersion())
	}
	reqVer, err := semver.NewVersion(headerVer)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid Infra-Version header: %v. Current version is %s", internal.ErrBadRequest, err, internal.FullVersion())
	}
	return reqVer, nil
}

func handlerForVersion(versions []routeVersion, reqVer *semver.Version) func(c *gin.Context) {
	if reqVer == nil {
		return nil
	}

	for _, v := range versions {
		if reqVer.GreaterThan(v.version) {
			continue
		}
		return v.handler
	}
	return nil
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
		handler:       handler,
		routeSettings: defaultRouteSettingsGet,
	})
}

var defaultRouteSettingsGet = routeSettings{
	omitFromTelemetry: true,
	txnOptions:        &sql.TxOptions{ReadOnly: true},
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

func readRequest(rCtx access.RequestContext, req interface{}) error {
	if len(c.Params) > 0 {
		params := make(map[string][]string)
		for _, v := range c.Params {
			params[v.Key] = []string{v.Value}
		}
		if err := binding.Uri.BindUri(params, req); err != nil {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
	}

	if len(c.Request.URL.Query()) > 0 {
		if err := binding.Query.Bind(c.Request, req); err != nil {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
	}

	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		if err := json.NewDecoder(c.Request.Body).Decode(req); err != nil {
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
		sendAPIError(c.Writer, c.Request, internal.ErrNotFound)
		return
	}

	c.Status(http.StatusNotFound)

	_, err := c.Writer.Write([]byte("404 not found"))
	if err != nil {
		logging.Errorf("%s", err.Error())
	}
}

func (a *API) deprecatedRoutes(noAuthnNoOrg *routeGroup) {
	// CLI clients before v0.14.4 rely on sign-up being false to continue with login
	type SignupEnabledResponse struct {
		Enabled bool `json:"enabled"`
	}

	add(a, noAuthnNoOrg, http.MethodGet, "/api/signup", route[api.EmptyRequest, *SignupEnabledResponse]{
		handler: func(rCtx access.RequestContext, _ *api.EmptyRequest) (*SignupEnabledResponse, error) {
			return &SignupEnabledResponse{Enabled: false}, nil
		},
		routeSettings: routeSettings{
			omitFromTelemetry: true,
			omitFromDocs:      true,
			txnOptions:        &sql.TxOptions{ReadOnly: true},
		},
	})
}
