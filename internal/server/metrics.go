package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/metrics"
)

var outboundRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "http",
	Name:      "outbound_request_duration_seconds",
	Help:      "A histogram of outbound call durations made from the server to an external source, in seconds.",
	Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
}, []string{"request_kind", "action"})

func setupMetrics(db *data.DB) *prometheus.Registry {
	registry := metrics.NewRegistry(productVersion())
	registry.MustRegister(collectors.NewDBStatsCollector(db.SQLdb(), "postgres"))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "users",
		Help:      "The total number of users",
	}, []string{}, func() []metrics.Metric {
		count, err := data.CountAllIdentities(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("users")
			return []metrics.Metric{}
		}

		return []metrics.Metric{
			{Count: float64(count)},
		}
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "groups",
		Help:      "The total number of groups",
	}, []string{}, func() []metrics.Metric {
		count, err := data.CountAllGroups(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("groups")
			return []metrics.Metric{}
		}

		return []metrics.Metric{
			{Count: float64(count)},
		}
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "grants",
		Help:      "The total number of grants",
	}, []string{}, func() []metrics.Metric {
		count, err := data.CountAllGrants(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("grants")
			return []metrics.Metric{}
		}

		return []metrics.Metric{
			{Count: float64(count)},
		}
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "providers",
		Help:      "The total number of providers",
	}, []string{"kind"}, func() []metrics.Metric {
		results, err := data.CountProvidersByKind(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("providers")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			values = append(values, metrics.Metric{
				Count:       result.Count,
				LabelValues: []string{result.Kind},
			})
		}

		return values
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "destinations",
		Help:      "The total number of destinations",
	}, []string{"version", "status"}, func() []metrics.Metric {
		results, err := data.CountDestinationsByConnectedVersion(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("destinations")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			status := "disconnected"
			if result.Connected {
				status = "connected"
			}

			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{result.Version, status}})
		}

		return values
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "organizations",
		Help:      "The total number of organizations",
	}, []string{}, func() []metrics.Metric {
		count, err := data.CountOrganizations(db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("organizations")
			return []metrics.Metric{}
		}

		return []metrics.Metric{
			{Count: float64(count)},
		}
	}))

	registry.MustRegister(outboundRequestDuration)

	return registry
}
