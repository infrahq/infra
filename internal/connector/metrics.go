package connector

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/metrics"
)

func setupMetrics() *prometheus.Registry {
	registry := metrics.NewRegistry()

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "build_info",
		Help: "A metric with a constant '1' value labeled by branch, version, commit, and date from which infra was built",
		ConstLabels: prometheus.Labels{
			"branch":  internal.Branch,
			"version": internal.FullVersion(),
			"commit":  internal.Commit,
			"date":    internal.Date,
		},
	}, func() float64 { return 1 }))

	return registry
}
