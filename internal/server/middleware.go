package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// TimeoutMiddleware adds a timeout to the request context within the Gin context.
// To correctly abort long-running requests, this depends on the users of the context to
// stop working when the context cancels.
// Note: The goroutine for the request is never halted; if the context is not
// passed down to lower packages and long-running tasks, then the app will not
// magically stop working on the request. No effort should be made to write
// an early http response here; it's up to the users of the context to watch for
// c.Request.Context().Err() or <-c.Request.Context().Done()
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func handleInfraDestinationHeader(rCtx access.RequestContext, uniqueID string) error {
	if uniqueID == "" {
		return nil
	}
	// not actually optional, but we already check that it's not an empty string
	destination, err := data.GetDestination(rCtx.DBTxn, data.ByOptionalUniqueID(uniqueID))
	switch {
	case errors.Is(err, internal.ErrNotFound):
		return nil // destination does not exist yet
	case err != nil:
		return err
	}

	// only save if there's significant difference between LastSeenAt and Now
	if time.Since(destination.LastSeenAt) > time.Second {
		destination.LastSeenAt = time.Now()
		if err := access.SaveDestination(rCtx, destination); err != nil {
			return fmt.Errorf("failed to update destination lastSeenAt: %w", err)
		}
	}
	return nil
}

// authenticateRequest is call for requests to routes that require authentication.
// It validates the access key and organization, updates the lastSeenAt of the user,
// and may also update the lastSeenAt of the destination if the appropriate header
// is set.
// authenticateRequest is also responsible for adding RequestContext to the
// gin.Context.
// See validateRequestOrganization for a related function used for unauthenticated
// routes.
func authenticateRequest(c *gin.Context, srv *Server) error {
	tx, err := srv.db.Begin(c.Request.Context())
	if err != nil {
		return err
	}
	defer logError(tx.Rollback, "failed to rollback middleware transaction")

	authned, err := requireAccessKey(c, tx, srv)
	if err != nil {
		return err
	}

	if _, err := validateOrgMatchesRequest(c.Request, tx, authned.Organization); err != nil {
		logging.L.Warn().Err(err).Msg("org validation failed")
		return internal.ErrBadRequest
	}
	tx = tx.WithOrgID(authned.Organization.ID)

	if uniqueID := c.Request.Header.Get("Infra-Destination"); uniqueID != "" {
		rCtx := access.RequestContext{DBTxn: tx, Authenticated: authned}
		if err := handleInfraDestinationHeader(rCtx, uniqueID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// TODO: move to caller
	rCtx := access.RequestContext{
		Request:       c.Request,
		Authenticated: authned,
	}
	c.Set(access.RequestContextKey, rCtx)
	return nil
}

// validateOrgMatchesRequest checks that if both the accessKeyOrg and the org
// from the request are set they have the same ID. If only one is set no
// error is returned.
//
// Returns the organization from any source that is not nil, or an error if the
// two sources do not match.
func validateOrgMatchesRequest(req *http.Request, tx data.GormTxn, accessKeyOrg *models.Organization) (*models.Organization, error) {
	orgFromRequest, err := getOrgFromRequest(req, tx)
	if err != nil {
		return nil, err
	}

	switch {
	case orgFromRequest == nil:
		return accessKeyOrg, nil
	case accessKeyOrg == nil:
		return orgFromRequest, nil
	case orgFromRequest.ID != accessKeyOrg.ID:
		return nil, fmt.Errorf("org from access key %v does not match org from request %v",
			accessKeyOrg.ID, orgFromRequest.ID)
	default:
		return orgFromRequest, nil
	}
}

// validateRequestOrganization is the alternative to authenticateRequest used
// for routes that don't require authentication. It checks for an optional
// access key, and if one does not exist, finds the organizationID from the
// hostname in the request.
//
// validateRequestOrganization is also responsible for adding RequestContext to the
// gin.Context.
func validateRequestOrganization(c *gin.Context, srv *Server) error {
	tx, err := srv.db.Begin(c.Request.Context())
	if err != nil {
		return err
	}
	defer logError(tx.Rollback, "failed to rollback middleware transaction")

	// ignore errors, access key is not required
	authned, _ := requireAccessKey(c, tx, srv)

	org, err := validateOrgMatchesRequest(c.Request, tx, authned.Organization)
	if err != nil {
		logging.L.Warn().Err(err).Msg("org validation failed")
		return internal.ErrBadRequest
	}

	// See this diagram for more details about this request flow
	// when an org is not specified.
	// https://github.com/infrahq/infra/blob/main/docs/dev/organization-request-flow.md

	// TODO: use an explicit setting for this, don't overload EnableSignup
	if org == nil && !srv.options.EnableSignup { // is single tenant
		org = srv.db.DefaultOrg
	}
	authned.Organization = org

	if err := tx.Commit(); err != nil {
		return err
	}

	// TODO: move to caller
	rCtx := access.RequestContext{
		Request:       c.Request,
		Authenticated: authned,
	}
	c.Set(access.RequestContextKey, rCtx)
	return nil
}

// requireAccessKey checks the bearer token is present and valid
func requireAccessKey(c *gin.Context, db *data.Transaction, srv *Server) (access.Authenticated, error) {
	var u access.Authenticated

	bearer, err := reqBearerToken(c, srv.options)
	if err != nil {
		return u, err
	}

	accessKey, err := data.ValidateRequestAccessKey(db, bearer)
	if err != nil {
		if errors.Is(err, data.ErrAccessKeyExpired) {
			return u, err
		}
		return u, fmt.Errorf("%w: invalid token: %s", internal.ErrUnauthorized, err)
	}

	if accessKey.Scopes.Includes(models.ScopePasswordReset) {
		// PUT /api/users/:id only
		if c.Request.URL.Path != "/api/users/"+accessKey.IssuedFor.String() || c.Request.Method != http.MethodPut {
			return u, fmt.Errorf("%w: temporary passwords can only be used to set new passwords", internal.ErrUnauthorized)
		}
	}

	org, err := data.GetOrganization(db, data.ByID(accessKey.OrganizationID))
	if err != nil {
		return u, fmt.Errorf("access key org lookup: %w", err)
	}

	// now that the org is loaded scope all db calls to that org
	// TODO: set the orgID explicitly in the options passed to GetIdentity to
	// remove the need for this WithOrgID.
	db = db.WithOrgID(org.ID)

	identity, err := data.GetIdentity(db, data.ByID(accessKey.IssuedFor))
	if err != nil {
		return u, fmt.Errorf("identity for access key: %w", err)
	}

	identity.LastSeenAt = time.Now().UTC()
	if err = data.SaveIdentity(db, identity); err != nil {
		return u, fmt.Errorf("identity update fail: %w", err)
	}

	u.AccessKey = accessKey
	u.Organization = org
	u.User = identity
	return u, nil
}

func getCookie(req *http.Request, name string) (string, error) {
	cookie, err := req.Cookie(name)
	if err != nil {
		return "", err
	}
	return url.QueryUnescape(cookie.Value)
}

func getRequestContext(c *gin.Context) access.RequestContext {
	raw, ok := c.Get(access.RequestContextKey)
	if !ok {
		return access.RequestContext{}
	}
	rCtx, ok := raw.(access.RequestContext)
	if !ok {
		return access.RequestContext{}
	}
	return rCtx
}

func getOrgFromRequest(req *http.Request, tx data.GormTxn) (*models.Organization, error) {
	host := req.Host

	logging.Debugf("Host: %s", host)
	if host == "" {
		return nil, nil
	}

hostLookup:
	org, err := data.GetOrganization(tx, data.ByDomain(host))
	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			logging.Debugf("Host not found: %s", host)
			// first, remove port and try again
			h, p, err := net.SplitHostPort(host)
			if len(p) > 0 && err == nil {
				host = h
				goto hostLookup
			}
			return nil, nil
		}
		return nil, err
	}
	return org, nil
}

func reqBearerToken(c *gin.Context, opts Options) (string, error) {
	header := c.Request.Header.Get("Authorization")

	bearer := ""

	parts := strings.Split(header, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		bearer = parts[1]
	} else {
		/*
		 Fallback to checking cookies.
		 The 'signup' cookie is set when a new org is created, check for it first.
		 Signup takes priority over the auth cookie to ensure a new signup always get the correct session.
		 If this isn't a new org, check for the 'auth' cookie which contains an access key.
		*/
		cookie := exchangeSignupCookieForSession(c, opts)
		if cookie == "" {
			logging.L.Trace().Msg("sign-up cookie not found, falling back to auth cookie")

			var err error
			cookie, err = getCookie(c.Request, cookieAuthorizationName)
			if err != nil {
				return "", fmt.Errorf("%w: valid token not found in request", internal.ErrUnauthorized)
			}
		}

		bearer = cookie
	}

	// this will get caught by key validation, but check to be safe
	if strings.TrimSpace(bearer) == "" {
		return "", fmt.Errorf("%w: skipped validating empty token", internal.ErrUnauthorized)
	}

	return bearer, nil
}

// logError calls fn and writes a log line at the warning level if the error is
// not nil. The log level is a warning because the error is not handled, which
// generally indicates the problem is not a critical error.
// logError accepts a function instead of an error so that it can be used with
// defer.
func logError(fn func() error, msg string) {
	if err := fn(); err != nil {
		logging.L.Warn().Err(err).Msg(msg)
	}
}
