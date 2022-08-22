package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/metrics"
)

func setupMetrics(db *gorm.DB) *prometheus.Registry {
	registry := metrics.NewRegistry(productVersion())

	if rawDB, err := db.DB(); err == nil {
		registry.MustRegister(collectors.NewDBStatsCollector(rawDB, db.Dialector.Name()))
	}

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "users",
		Help:      "The total number of users",
	}, []string{}, func() []metrics.Metric {
		count, err := data.GlobalCount[models.Identity](db, data.NotName("connector"))
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
		count, err := data.GlobalCount[models.Group](db)
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
		count, err := data.GlobalCount[models.Grant](db, data.NotPrivilege("connector"))
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
			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{result.Kind}})
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
		count, err := data.GlobalCount[models.Organization](db)
		if err != nil {
			logging.L.Warn().Err(err).Msg("organizations")
			return []metrics.Metric{}
		}

		return []metrics.Metric{
			{Count: float64(count)},
		}
	}))

	return registry
}
