package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "http",
	Name:      "request_duration_seconds",
	Help:      "A histogram of duration, in seconds, handling HTTP requests.",
	Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
}, []string{"host", "method", "path", "status"})

// Middleware registers a request_duration_seconds metric with promRegistry
// and returns a middleware that emits the metric on every request.
func Middleware(promRegistry prometheus.Registerer) gin.HandlerFunc {
	promRegistry.MustRegister(requestDuration)
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
