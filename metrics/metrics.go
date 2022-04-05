package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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

		RequestDuration.With(prometheus.Labels{
			"host":   c.Request.Host,
			"method": c.Request.Method,
			"path":   c.FullPath(),
			"status": strconv.Itoa(c.Writer.Status()),
		}).Observe(time.Since(t).Seconds())
	}
}

// NewHandler creates a new gin.Engine, and adds a 'GET /metrics' handler to it.
// The handler serves prometheus metrics from the promRegistry.
func NewHandler(promRegistry *prometheus.Registry) *gin.Engine {
	engine := gin.New()
	engine.GET("/metrics", func(c *gin.Context) {
		handler := promhttp.InstrumentMetricHandler(
			promRegistry,
			promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
		handler.ServeHTTP(c.Writer, c.Request)
	})
	return engine
}
