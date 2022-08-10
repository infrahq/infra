package server

import (
	"context"
	"errors"
	"fmt"
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
func authenticatedMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		withDBTxn(c.Request.Context(), db, func(tx *gorm.DB) {

			// TODO: get this from server
			org, err := data.GetOrganization(tx, data.ByName(models.DefaultOrganizationName))
			if err != nil {
				sendAPIError(c, err)
				return
			}
			c.Request = c.Request.WithContext(data.WithOrg(c.Request.Context(), org))
			tx.Statement.Context = c.Request.Context() // TODO: remove with gorm

			authned, err := requireAccessKey(tx, c.Request)
			if err != nil {
				sendAPIError(c, err)
				return
			}

			rCtx := access.RequestContext{
				Request:       c.Request,
				DBTxn:         tx,
				Authenticated: authned,
			}
			c.Set(access.RequestContextKey, rCtx)

			// TODO: remove once everything uses RequestContext
			c.Set("identity", authned.User)
			c.Set("db", tx)

			if err := handleInfraDestinationHeader(c); err != nil {
				sendAPIError(c, err)
				return
			}
			c.Next()
		})
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

func unauthenticatedMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		withDBTxn(c.Request.Context(), db, func(tx *gorm.DB) {
			// TODO: get this from server
			org, err := data.GetOrganization(tx, data.ByName(models.DefaultOrganizationName))
			if err != nil {
				sendAPIError(c, err)
				return
			}
			c.Request = c.Request.WithContext(data.WithOrg(c.Request.Context(), org))
			tx.Statement.Context = c.Request.Context() // TODO: remove with gorm

			rCtx := access.RequestContext{
				Request: c.Request,
				DBTxn:   tx,
			}
			c.Set(access.RequestContextKey, rCtx)
			// TODO: remove once everything uses RequestContext
			c.Set("db", tx)
			c.Next()
		})
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
