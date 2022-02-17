package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func SetupMetrics(db *gorm.DB) error {
	promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "build",
		Name:      "info",
		Help:      "Build information about Infra Server.",
	}, []string{"branch", "version", "commit", "date"}).With(prometheus.Labels{
		"branch":  internal.Branch,
		"version": internal.Version,
		"commit":  internal.Commit,
		"date":    internal.Date,
	}).Set(1)

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "users",
		Help:      "Number of users managed by Infra.",
	}, func() float64 {
		count, err := data.Count[models.User](db)
		if err != nil {
			logging.S.Warnf("users: %w", err)
			return 0
		}

		return float64(*count)
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "groups",
		Help:      "Number of groups managed by Infra.",
	}, func() float64 {
		count, err := data.Count[models.Group](db)
		if err != nil {
			logging.S.Warnf("groups: %w", err)
			return 0
		}

		return float64(*count)
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "grants",
		Help:      "Number of grants managed by Infra.",
	}, func() float64 {
		count, err := data.Count[models.Grant](db)
		if err != nil {
			logging.S.Warnf("grants: %w", err)
			return 0
		}

		return float64(*count)
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "providers",
		Help:      "Number of providers managed by Infra.",
	}, func() float64 {
		count, err := data.Count[models.Provider](db)
		if err != nil {
			logging.S.Warnf("providers: %w", err)
			return 0
		}

		return float64(*count)
	})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "destinations",
		Help:      "Number of destinations managed by Infra.",
	}, func() float64 {
		count, err := data.Count[models.Destination](db)
		if err != nil {
			logging.S.Warnf("destinations: %w", err)
			return 0
		}

		return float64(*count)
	})

	promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "database",
		Name:      "info",
		Help:      "Information about configured database.",
	}, []string{"name"}).With(prometheus.Labels{
		"name": db.Dialector.Name(),
	}).Set(1)

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "database",
		Name:      "connected",
		Help:      "Database connection status.",
	}, func() float64 {
		pinger, ok := db.ConnPool.(interface{ Ping() error })
		if !ok {
			logging.L.Warn("ping: not supported")
			return -1
		}

		if err := pinger.Ping(); err != nil {
			logging.L.Warn("ping: not connected")
			return 0
		}

		return 1
	})

	return nil
}
