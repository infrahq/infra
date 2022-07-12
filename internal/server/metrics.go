package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/metrics"
)

func setupMetrics(db *gorm.DB) *prometheus.Registry {
	registry := metrics.NewRegistry()

	if rawDB, err := db.DB(); err == nil {
		registry.MustRegister(collectors.NewDBStatsCollector(rawDB, db.Dialector.Name()))
	}

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "users",
		Help:      "The total number of users",
	}, []string{}, func() []metrics.Metric {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM identities WHERE deleted_at IS NULL").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("users")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "groups",
		Help:      "The total number of groups",
	}, []string{}, func() []metrics.Metric {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM groups WHERE deleted_at IS NULL").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("groups")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "grants",
		Help:      "The total number of grants",
	}, []string{}, func() []metrics.Metric {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM grants WHERE deleted_at IS NULL").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("grants")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(metrics.NewCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "providers",
		Help:      "The total number of providers",
	}, []string{"kind"}, func() []metrics.Metric {
		var results []struct {
			Kind  string
			Count int
		}

		if err := db.Raw("SELECT kind, COUNT(*) as count FROM providers WHERE deleted_at IS NULL GROUP BY kind").Scan(&results).Error; err != nil {
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
	}, []string{"version"}, func() []metrics.Metric {
		var results []struct {
			Version string
			Count   int
		}

		if err := db.Raw("SELECT COALESCE(version, '') AS version, COUNT(*) as count FROM destinations WHERE deleted_at IS NULL GROUP BY COALESCE(version, '')").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("destinations")
			return []metrics.Metric{}
		}

		values := make([]metrics.Metric, 0, len(results))
		for _, result := range results {
			values = append(values, metrics.Metric{Count: float64(result.Count), LabelValues: []string{result.Version}})
		}

		return values
	}))

	return registry
}
