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
	"gorm.io/gorm"

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

func handleInfraDestinationHeader(c *gin.Context) error {
	uniqueID := c.Request.Header.Get("Infra-Destination")
	if uniqueID == "" {
		return nil
	}

	// TODO: use GetDestination(ByUniqueID())
	destinations, err := access.ListDestinations(c, uniqueID, "", &models.Pagination{Limit: 1})
	if err != nil {
		return err
	}

	switch len(destinations) {
	case 0:
		// destination does not exist yet, noop
		return nil
	case 1:
		destination := destinations[0]
		// only save if there's significant difference between LastSeenAt and Now
		if time.Since(destination.LastSeenAt) > time.Second {
			destination.LastSeenAt = time.Now()
			if err := access.SaveDestination(c, &destination); err != nil {
				return fmt.Errorf("failed to update destination lastSeenAt: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("multiple destinations found for unique ID %q", uniqueID)
	}
}

// authenticatedMiddleware is applied to all routes that require authentication.
// It validates the access key, and updates the lastSeenAt of the user, and
// possibly also of the destination.
func authenticatedMiddleware(srv *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		withDBTxn(c.Request.Context(), srv.DB().GormDB(), func(db *gorm.DB) {
			tx := data.NewTransaction(db, 0)
			authned, err := requireAccessKey(c, tx, srv)
			if err != nil {
				sendAPIError(c, err)
				return
			}

			if _, err := validateOrgMatchesRequest(c.Request, tx, authned.Organization); err != nil {
				logging.L.Warn().Err(err).Msg("org validation failed")
				sendAPIError(c, internal.ErrBadRequest)
				return
			}

			tx = data.NewTransaction(db, authned.Organization.ID)
			rCtx := access.RequestContext{
				Request:       c.Request,
				DBTxn:         tx,
				Authenticated: authned,
			}
			c.Set(access.RequestContextKey, rCtx)

			// TODO: remove once everything uses RequestContext
			c.Set("identity", authned.User)

			if err := handleInfraDestinationHeader(c); err != nil {
				sendAPIError(c, err)
				return
			}
			c.Next()
		})
	}
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

func withDBTxn(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB)) {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fn(tx)
		return nil
	})
	// TODO: https://github.com/infrahq/infra/issues/2697
	if err != nil {
		logging.L.Error().Err(err).Msg("failed to commit database transaction")
	}
}

func unauthenticatedMiddleware(srv *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		withDBTxn(c.Request.Context(), srv.DB().GormDB(), func(db *gorm.DB) {
			tx := data.NewTransaction(db, 0)
			// ignore errors, access key is not required
			authned, _ := requireAccessKey(c, tx, srv)

			org, err := validateOrgMatchesRequest(c.Request, tx, authned.Organization)
			if err != nil {
				logging.L.Warn().Err(err).Msg("org validation failed")
				sendAPIError(c, internal.ErrBadRequest)
				return
			}
			authned.Organization = org

			// See this diagram for more details about this request flow
			// when an org is not specified.
			// https://github.com/infrahq/infra/blob/main/docs/dev/organization-request-flow.md

			// TODO: use an explicit setting for this, don't overload EnableSignup
			if org == nil && !srv.options.EnableSignup { // is single tenant
				org = srv.db.DefaultOrg
			}
			if org != nil {
				authned.Organization = org
				tx = data.NewTransaction(db, org.ID)
			}

			rCtx := access.RequestContext{
				Request:       c.Request,
				DBTxn:         tx,
				Authenticated: authned,
			}
			c.Set(access.RequestContextKey, rCtx)
			c.Next()
		})
	}
}

func orgRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		rCtx := getRequestContext(c)

		if rCtx.Authenticated.Organization == nil {
			sendAPIError(c, internal.ErrBadRequest)
			return
		}
		c.Next()
	}
}

// requireAccessKey checks the bearer token is present and valid
func requireAccessKey(c *gin.Context, db data.GormTxn, srv *Server) (access.Authenticated, error) {
	var u access.Authenticated

	bearer, err := reqBearerToken(c, srv.options.BaseDomain)
	if err != nil {
		return u, err
	}

	accessKey, err := data.ValidateAccessKey(db, bearer)
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
	db = data.NewTransaction(db.GormDB(), org.ID)

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

func reqBearerToken(c *gin.Context, baseDomain string) (string, error) {
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
		cookie, err := getCookie(c.Request, cookieSignupName)
		if err == nil {
			exchangeSignupCookieForSession(c, baseDomain)
		} else {
			logging.L.Trace().Err(err).Msg("sign-up cookie not found, falling back to auth cookie")

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
