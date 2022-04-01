package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
)

var requestTimeout = 60 * time.Second

// RequestTimeoutMiddleware adds a timeout to the request context within the Gin context.
// To correctly abort long-running requests, this depends on the users of the context to
// stop working when the context cancels.
// Note: The goroutine for the request is never halted; if the context is not
// passed down to lower packages and long-running tasks, then the app will not
// magically stop working on the request. No effort should be made to write
// an early http response here; it's up to the users of the context to watch for
// c.Request.Context().Err() or <-c.Request.Context().Done()
func RequestTimeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		start := time.Now()

		c.Next()

		if elapsed := time.Since(start); elapsed > requestTimeout {
			logging.L.Sugar().Warnf("Request to %q took %s and may have timed out", c.Request.URL.Path, elapsed)
		}
	}
}

// DatabaseMiddleware injects a `db` object into the Gin context.
func DatabaseMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := db.Transaction(func(tx *gorm.DB) error {
			c.Set("db", tx)
			c.Next()
			return nil
		})
		if err != nil {
			logging.S.Debugf(err.Error())
		}
	}
}

// AuthenticationMiddleware validates the incoming token
func AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := RequireAccessKey(c); err != nil {
			logging.S.Debug(err.Error())
			// IMPORTANT: do not return errors encountered during token validation, always return generic unauthorized message
			sendAPIError(c, internal.ErrUnauthorized)

			return
		}

		c.Next()
	}
}

// RequireAccessKey checks the bearer token is present and valid
func RequireAccessKey(c *gin.Context) error {
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		return errors.New("unknown db type in context")
	}

	header := c.Request.Header.Get("Authorization")

	bearer := ""

	parts := strings.Split(header, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		bearer = parts[1]
	} else {
		// Fall back to checking cookies
		cookie, err := c.Cookie(CookieAuthorizationName)
		if err != nil {
			return fmt.Errorf("%w: valid token not found in request", internal.ErrUnauthorized)
		}

		bearer = cookie
	}

	// this will get caught by key validation, but check to be safe
	if strings.TrimSpace(bearer) == "" {
		return fmt.Errorf("%w: skipped validating empty token", internal.ErrUnauthorized)
	}

	accessKey, err := data.ValidateAccessKey(db, bearer)
	if err != nil {
		return fmt.Errorf("%w: invalid token: %s", internal.ErrUnauthorized, err)
	}

	c.Set("key", accessKey)

	identity, err := data.GetIdentity(db, data.ByID(accessKey.IssuedFor))
	if err != nil {
		return fmt.Errorf("identity for token: %w", err)
	}

	identity.LastSeenAt = time.Now().UTC()
	if err = data.SaveIdentity(db, identity); err != nil {
		return fmt.Errorf("%w: identity update fail: %s", internal.ErrUnauthorized, err)
	}

	c.Set("identity", identity)

	return nil
}
