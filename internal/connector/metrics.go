package connector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/infrahq/infra/internal"
)

func setupMetrics() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "build",
		Name:      "info",
		Help:      "Build information about Infra Connector.",
	}, []string{"branch", "version", "commit", "date"}).With(prometheus.Labels{
		"branch":  internal.Branch,
		"version": internal.FullVersion(),
		"commit":  internal.Commit,
		"date":    internal.Date,
	}).Set(1)

	return registry
}
