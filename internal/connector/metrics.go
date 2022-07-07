package connector

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/infrahq/infra/internal"
)

func setupMetrics() *prometheus.Registry {
	registry := prometheus.NewRegistry()

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
