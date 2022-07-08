package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "http",
	Name:      "request_duration_seconds",
	Help:      "A histogram of duration, in seconds, handling HTTP requests.",
	Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
}, []string{"host", "method", "path", "status"})

// Middleware registers metrics with promRegistry and returns a middleware that
// emits a request_duration_seconds metric on every request.
//
// The metrics registered with the registry include:
//   * the standard process metrics
//   * the standard go metrics
//   * the request_duration_seconds metric emitted by the middleware
func Middleware(promRegistry prometheus.Registerer) gin.HandlerFunc {
	promRegistry.MustRegister(requestDuration)
	promRegistry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	promRegistry.MustRegister(collectors.NewGoCollector())

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

// Metric is a container for a count metric and its related labels
type Metric struct {
	Count       float64
	LabelValues []string
}

// Collector implements the prometheus.Collector interface. It creates a Prometheus Metric for each
// item returned by collectFunc and sets the count and label values accordingly.
type Collector struct {
	desc        *prometheus.Desc
	collectFunc func() []Metric
}

// NewCollector creates a Collector
func NewCollector(opts prometheus.Opts, labelNames []string, collectFunc func() []Metric) *Collector {
	fqname := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)
	return &Collector{
		desc:        prometheus.NewDesc(fqname, opts.Help, labelNames, opts.ConstLabels),
		collectFunc: collectFunc,
	}
}

// Describe is implemented by DescribeByCollect
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect implements Collector. It create a set of constant metrics with the values and labels
// as described by collectFunc
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range c.collectFunc() {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, float64(metric.Count), metric.LabelValues...)
	}
}
