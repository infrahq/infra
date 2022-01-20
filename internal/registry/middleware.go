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
	"github.com/infrahq/infra/internal/registry/models"
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
		if err := RequireAuthentication(c); err != nil {
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

		labels := prometheus.Labels{
			"method":  c.Request.Method,
			"handler": path,
		}

		requestInProgressGauge.With(labels).Inc()

		c.Next()

		requestInProgressGauge.With(labels).Dec()

		labels["status"] = strconv.Itoa(c.Writer.Status())

		requestCount.With(labels).Inc()
		requestDuration.With(labels).Observe(time.Since(t).Seconds())
	}
}

// RequireAuthentication checks the bearer token is present and valid then adds its permissions to the context
func RequireAuthentication(c *gin.Context) error {
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

	if len(bearer) != models.TokenLength {
		return fmt.Errorf("%w: rejected token of invalid length", internal.ErrUnauthorized)
	}

	token, err := data.GetToken(db, data.ByKey(bearer[:models.TokenKeyLength]))
	if err != nil {
		return fmt.Errorf("%w could not get token from database, it may not exist: %s", internal.ErrUnauthorized, err)
	}

	if err := data.CheckTokenSecret(token, bearer); err != nil {
		return fmt.Errorf("%w: rejected invalid token: %s", internal.ErrUnauthorized, err)
	}

	if err := data.CheckTokenExpired(token); err != nil {
		return fmt.Errorf("%w: rejected token: %s", internal.ErrUnauthorized, err)
	}

	c.Set("authentication", bearer)

	// token is valid, check where to set permissions from
	if token.UserID != 0 {
		// this token has a parent user, set by their current permissions
		user, err := data.GetUser(db, data.ByID(token.UserID))
		if err != nil {
			return fmt.Errorf("%w: %s", internal.ErrUnauthorized, err)
		}

		c.Set("permissions", user.Permissions)
		c.Set("user_id", token.UserID)
		logging.S.Debug("user permissions: %s \n", user.Permissions)

		user.LastSeenAt = time.Now()
		if err := data.UpdateUser(db, user, data.ByID(user.ID)); err != nil {
			return fmt.Errorf("%w: user update fail: %s", internal.ErrUnauthorized, err)
		}

		c.Set("user", user)
	} else if token.APITokenID != 0 {
		// this is an API token
		apiToken, err := data.GetAPIToken(db, data.ByID(token.APITokenID))
		if err != nil {
			return fmt.Errorf("%w: %s", internal.ErrUnauthorized, err)
		}
		c.Set("permissions", apiToken.Permissions)
	}

	return nil
}
