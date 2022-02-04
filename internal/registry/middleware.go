package registry

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/data"
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

// AuthenticationMiddleware validates the incoming token and adds their permissions to the context
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

// MetricsMiddleware wraps the request with a standard set of Prometheus metrics.
// It has an additional responsibility of stripping out any unique identifiers as it will
// drastically increase the cardinality, and cost, of produced metrics.
func MetricsMiddleware(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		c.Next()

		requestDuration.With(prometheus.Labels{
			"method":  c.Request.Method,
			"handler": path,
			"status":  strconv.Itoa(c.Writer.Status()),
		}).Observe(time.Since(t).Seconds())
	}
}

// RequireAccessKey checks the bearer token is present and valid then adds its permissions to the context
func RequireAccessKey(c *gin.Context) error {
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		return errors.New("unknown db type in context")
	}

	header := c.Request.Header.Get("Authorization")

	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return fmt.Errorf("%w: valid token not found in authorization header, expecting the format `Bearer $token`", internal.ErrUnauthorized)
	}

	bearer := parts[1]

	token, err := data.LookupAccessKey(db, bearer)
	if err != nil {
		return fmt.Errorf("%w: invalid token: %s", internal.ErrUnauthorized, err)
	}

	c.Set("token", token)

	// token is valid, check where to set permissions from
	if token.UserID != 0 {
		// this token has a parent user, set by their current permissions
		user, err := data.GetUser(db, data.ByID(token.UserID))
		if err != nil {
			return fmt.Errorf("user for token: %w", err)
		}

		c.Set("permissions", user.Permissions)
		logging.S.Debugf("user permissions: %s", user.Permissions)

		user.LastSeenAt = time.Now()
		if err = data.SaveUser(db, user); err != nil {
			return fmt.Errorf("%w: user update fail: %s", internal.ErrUnauthorized, err)
		}

		c.Set("user", user)
	}

	if len(token.Permissions) > 0 {
		c.Set("permissions", token.Permissions)
	}

	return nil
}
