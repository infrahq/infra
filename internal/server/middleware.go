package server

import (
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
	"github.com/infrahq/infra/internal/server/redis"
)

func handleInfraDestinationHeader(rCtx access.RequestContext, uniqueID string) error {
	if uniqueID == "" {
		return nil
	}
	destination, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByUniqueID: uniqueID})
	switch {
	case errors.Is(err, internal.ErrNotFound):
		return nil // destination does not exist yet
	case err != nil:
		return err
	}

	// only save if there's significant difference between LastSeenAt and Now
	if time.Since(destination.LastSeenAt) > lastSeenUpdateThreshold {
		destination.LastSeenAt = time.Now()
		if err := access.SaveDestination(rCtx, destination); err != nil {
			return fmt.Errorf("failed to update destination lastSeenAt: %w", err)
		}
	}
	return nil
}

// authenticateRequest authenticates the user performing the request.
//
// If the route requires authentication, authenticateRequest validates the access
// key and organization, updates the lastSeenAt of the user,
// and may also update the lastSeenAt of the destination if the appropriate header
// is set.
//
// If the route does not require authentication, authenticateRequest will attempt
// the same authentication, but ignore the error. It will also look up the
// organization from the domain name when no access key is provided.
//
// If the request identifies an organization (which is required for most routes)
// a rate limit will be applied to all requests from the same organization.
func authenticateRequest(c *gin.Context, route routeSettings, srv *Server) (access.Authenticated, error) {
	tx, err := srv.db.Begin(c.Request.Context(), nil)
	if err != nil {
		return access.Authenticated{}, err
	}
	defer logError(tx.Rollback, "failed to rollback middleware transaction")

	authned, err := requireAccessKey(c, tx, srv)
	if !route.authenticationOptional && err != nil {
		return authned, err
	}

	org, err := validateOrgMatchesRequest(c.Request, tx, authned.Organization)
	if err != nil {
		return authned, err
	}

	if route.authenticationOptional {
		// See this diagram for more details about this request flow
		// when an org is not specified.
		// https://github.com/infrahq/infra/blob/main/docs/dev/organization-request-flow.md

		// TODO: use an explicit setting for this, don't overload EnableSignup
		if org == nil && !srv.options.EnableSignup { // is single tenant
			org = srv.db.DefaultOrg
		}
		if org == nil && !route.organizationOptional {
			return authned, fmt.Errorf("%w: missing organization", internal.ErrBadRequest)
		}
		authned.Organization = org
	}

	if org != nil {
		// TODO: limit should be a per-organization setting
		if err := redis.NewLimiter(srv.redis).RateOK(org.ID.String(), 5000); err != nil {
			return authned, err
		}
	}

	if authned.User != nil {
		if uniqueID := c.Request.Header.Get(headerInfraDestination); uniqueID != "" {
			tx = tx.WithOrgID(authned.Organization.ID)
			rCtx := access.RequestContext{DBTxn: tx, Authenticated: authned}
			if err := handleInfraDestinationHeader(rCtx, uniqueID); err != nil {
				return authned, err
			}
		}
	}

	err = tx.Commit()
	return authned, err
}

// lastSeenUpdateThreshold is the duration of time that must pass before a
// LastSeenAt value for a user or destination is updated again. This prevents
// excessive writes when a single user performs many requests in a short
// period of time.
//
// If you change this value, you may also want to change the threshold in
// data.ValidateRequestAccessKey.
const lastSeenUpdateThreshold = 2 * time.Second

const headerInfraDestination = "Infra-Destination"

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
		return nil, fmt.Errorf("%w: org from access key %v does not match org from request %v",
			internal.ErrBadRequest, accessKeyOrg.ID, orgFromRequest.ID)
	default:
		return orgFromRequest, nil
	}
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

	identity, err := data.GetIdentity(db, data.GetIdentityOptions{ByID: accessKey.IssuedFor})
	if err != nil {
		return u, fmt.Errorf("identity for access key: %w", err)
	}

	if time.Since(identity.LastSeenAt) > lastSeenUpdateThreshold {
		identity.LastSeenAt = time.Now().UTC()
		if err = data.UpdateIdentity(db, identity); err != nil {
			return u, fmt.Errorf("identity update fail: %w", err)
		}
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
