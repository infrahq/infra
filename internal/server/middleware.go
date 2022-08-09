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

// DatabaseMiddleware injects a `db` object into the Gin context.
func DatabaseMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			tx.Statement.Context = context.WithValue(tx.Statement.Context, data.OrgCtxKey{}, db.Statement.Context.Value(data.OrgCtxKey{}))
			c.Set("db", tx)
			c.Next()
			return nil
		})
		if err != nil {
			logging.Debugf(err.Error())
			sendAPIError(c, err) // TODO: is this going to work if something lower down the stack wrote the response already?
			_ = c.Error(err)
		}
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

// TODO: remove duplicate in access package
func getDB(c *gin.Context) *gorm.DB {
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		return nil
	}
	return db
}

// authenticatedMiddleware is applied to all routes that require authentication.
// It validates the access key, and updates the lastSeenAt of the user, and
// possibly also of the destination.
func authenticatedMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := getDB(c)
		authned, err := requireAccessKey(db, c.Request)
		if err != nil {
			sendAPIError(c, err)
			return
		}

		rCtx := access.RequestContext{
			Request:       c.Request,
			DBTxn:         db,
			Authenticated: authned,
		}
		c.Set(access.RequestContextKey, rCtx)

		// TODO: remove
		c.Set("identity", authned.User)

		if err := handleInfraDestinationHeader(c); err != nil {
			sendAPIError(c, err)
			return
		}
		c.Next()
	}
}

// requireAccessKey checks the bearer token is present and valid
func requireAccessKey(db *gorm.DB, req *http.Request) (access.Authenticated, error) {
	var u access.Authenticated
	header := req.Header.Get("Authorization")

	bearer := ""

	parts := strings.Split(header, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		bearer = parts[1]
	} else {
		// Fall back to checking cookies
		cookie, err := getCookie(req, cookieAuthorizationName)
		if err != nil {
			return u, fmt.Errorf("%w: valid token not found in request", internal.ErrUnauthorized)
		}

		bearer = cookie
	}

	// this will get caught by key validation, but check to be safe
	if strings.TrimSpace(bearer) == "" {
		return u, fmt.Errorf("%w: skipped validating empty token", internal.ErrUnauthorized)
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
		if req.URL.Path != "/api/users/"+accessKey.IssuedFor.String() || req.Method != http.MethodPut {
			return u, fmt.Errorf("%w: temporary passwords can only be used to set new passwords", internal.ErrUnauthorized)
		}
	}

	identity, err := data.GetIdentity(db, data.ByID(accessKey.IssuedFor))
	if err != nil {
		return u, fmt.Errorf("identity for token: %w", err)
	}

	identity.LastSeenAt = time.Now().UTC()
	if err = data.SaveIdentity(db, identity); err != nil {
		return u, fmt.Errorf("identity update fail: %w", err)
	}

	u.AccessKey = accessKey
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

func OrganizationFromDomain(defaultOrgName, defaultOrgDomain string) gin.HandlerFunc {
	return func(c *gin.Context) {
		org, host, err := getOrgFromRequest(c, defaultOrgName, defaultOrgDomain)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		if org != nil {
			c.Set("host", host)
			c.Set("org", org)
			// c.Request.WithContext(context.WithValue(c.Request.Context(), "org", org))
			db := getDB(c)
			db.Statement.Context = context.WithValue(db.Statement.Context, data.OrgCtxKey{}, org)
			c.Set("db", db)
			logging.Debugf("organization set to %s for host %s", org.Name, host)
		}
		c.Next()
	}
}

// returns org, host
func getOrgFromRequest(c *gin.Context, defaultOrgName, defaultOrgDomain string) (*models.Organization, string, error) {
	db := getDB(c)
	host := c.Request.Header.Get("Host")

	if len(host) == 0 {
		// tests are lazy and don't set Host
		host = defaultOrgDomain
	}

	if len(host) > 0 {
	hostLookup:
		org, err := data.GetOrganization(db, data.ByDomain(host))
		if err != nil {
			if errors.Is(err, internal.ErrNotFound) {
				// first, remove port and try again
				h, p, err := net.SplitHostPort(host)
				if len(p) > 0 && err == nil {
					host = h
					goto hostLookup
				}

				org, err = getDefaultOrg(db, defaultOrgName, defaultOrgDomain)
				if err != nil {
					return nil, "", fmt.Errorf("creating default organization: %w", err)
				}
				return org, host, nil
			}
			logging.Errorf("fetching org: %s", err)
			return nil, "", nil
		}
		return org, host, nil
	}
	return nil, "", nil
}

func OrganizationRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		o, ok := c.Get("org")
		if !ok {
			sendAPIError(c, internal.ErrBadRequest)
			return
		}
		if _, ok = o.(*models.Organization); !ok {
			sendAPIError(c, internal.ErrBadRequest)
			return
		}
		c.Next()
	}
}

var defaultOrgCache *models.Organization

// check for configured default org
func getDefaultOrg(db *gorm.DB, defaultOrgName, defaultOrgDomain string) (*models.Organization, error) {
	if defaultOrgCache != nil {
		return defaultOrgCache, nil
	}
	if len(defaultOrgName) == 0 {
		return nil, errors.New("organization not configured")
	}
	org, err := data.GetOrganization(db, data.ByName(defaultOrgName))
	if err != nil {
		return nil, err
	}

	defaultOrgCache = org
	return org, nil
}

func unauthenticatedMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rCtx := access.RequestContext{
			Request: c.Request,
			DBTxn:   getDB(c),
		}
		c.Set(access.RequestContextKey, rCtx)
	}
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
