package registry

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
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
		if err := access.RequireAuthentication(c); err != nil {
			logging.S.Debug(err.Error())
			// IMPORTANT: do not return errors encountered during token validation, always return generic unauthorized message
			sendAPIError(c, http.StatusUnauthorized, internal.ErrInvalid)

			return
		}

		c.Next()
	}
}

// MetricsMiddleware wraps the request with a standard set of Prometheus metrics.
// It has an additional responsibility of stripping out any unique identifiers as it will
// drastically increase the cardinality, and cost, of produced metrics.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		path := make([]string, 0)

		parts := strings.Split(c.Request.URL.Path, "/")
		for _, part := range parts {
			if _, err := uuid.Parse(part); err != nil {
				path = append(path, part)
			} else {
				path = append(path, ":id")
			}
		}

		labels := prometheus.Labels{
			"method":  c.Request.Method,
			"handler": strings.Join(path, "/"),
		}

		requestInProgressGauge.With(labels).Inc()

		c.Next()

		requestInProgressGauge.With(labels).Dec()

		labels["status"] = strconv.Itoa(c.Writer.Status())

		requestCount.With(labels).Inc()
		requestDuration.With(labels).Observe(time.Since(t).Seconds())
	}
}
