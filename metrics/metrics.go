package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
)

// NewRegistry creates a new prometheus.Registry and registers common collectors and metrics.
//
// NewRegistry registers:
//   - standard process collector
//   - standard go collector
//   - build_info metric
func NewRegistry(version string) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "build_info",
		Help: "A metric with a constant '1' value labeled by branch, version, commit, and date from which infra was built",
		ConstLabels: prometheus.Labels{
			"branch":  internal.Branch,
			"version": version,
			"commit":  internal.Commit,
			"date":    internal.Date,
		},
	}, func() float64 { return 1 }))

	return registry
}

var providerMetricsKey = "provider-metrics"

func ProviderCallMetricsFromContext(c context.Context) *prometheus.HistogramVec {
	if raw := c.Value(providerMetricsKey); raw != nil {
		return raw.(*prometheus.HistogramVec) // nolint:forcetypeassert
	}
	return nil
}

type ProviderActionOpts struct {
	ProviderID      uid.ID
	OrgID           uid.ID
	Action          string
	DurationSeconds float64
}

func ProviderAction(c context.Context, opts ProviderActionOpts) {
	m := ProviderCallMetricsFromContext(c)
	if m != nil {
		m.With(prometheus.Labels{
			"provider_id":     opts.ProviderID.String(),
			"organization_id": opts.OrgID.String(),
			"action":          opts.Action,
		}).Observe(opts.DurationSeconds)
	}
}

// Middleware registers the http_request_duration_seconds histogram metric with registry
// and returns a middleware that emits a request_duration_seconds metric on every request.
func Middleware(registry prometheus.Registerer) gin.HandlerFunc {
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http",
		Name:      "request_duration_seconds",
		Help:      "A histogram of duration, in seconds, handling HTTP requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"host", "method", "path", "status", "blocking"})

	requestCount := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "http",
		Name:      "requests_active",
		Help:      "A gauge of the number of requests currently being handled.",
	}, []string{"blocking"})

	outboundCallDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http",
		Name:      "outbound_request_call_seconds",
		Help:      "A histogram of outbound call durations made from the server to an external source, in seconds.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"provider_id", "organization_id", "action"})

	registry.MustRegister(requestDuration, requestCount)

	return func(c *gin.Context) {
		blocking := blockingRequestLabel(c.Request)

		count := requestCount.With(prometheus.Labels{"blocking": blocking})
		count.Inc()
		defer count.Dec()

		metricContext := context.WithValue(c.Request.Context(), providerMetricsKey, outboundCallDuration)
		c.Request = c.Request.WithContext(metricContext)

		t := time.Now()
		c.Next()

		requestDuration.With(prometheus.Labels{
			"host":     c.Request.Host,
			"method":   c.Request.Method,
			"path":     c.FullPath(),
			"status":   strconv.Itoa(c.Writer.Status()),
			"blocking": blocking,
		}).Observe(time.Since(t).Seconds())
	}
}

func blockingRequestLabel(req *http.Request) string {
	switch req.URL.Query().Get("lastUpdateIndex") {
	case "", "0":
		return "false"
	default:
		return "true"
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
