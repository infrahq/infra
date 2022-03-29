package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "http",
	Name:      "request_duration_seconds",
	Help:      "A histogram of duration, in seconds, handling HTTP requests.",
	Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
}, []string{"host", "method", "path", "status"})

// Middleware wraps the request with a standard set of Prometheus metrics.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		c.Next()

		requestDuration.With(prometheus.Labels{
			"host":   c.Request.Host,
			"method": c.Request.Method,
			"path":   c.FullPath(),
			"status": strconv.Itoa(c.Writer.Status()),
		}).Observe(time.Since(t).Seconds())
	}
}
