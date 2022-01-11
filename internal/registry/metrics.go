package registry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

var (
	requestInProgressGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "http",
		Name:      "requests_in_progress",
		Help:      "Number of HTTP requests currently in progress.",
	}, []string{"method", "handler"})

	requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests served.",
	}, []string{"method", "handler", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http",
		Name:      "requests_duration_seconds",
		Help:      "A histogram of the duration, in seconds, handling HTTP requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"method", "handler", "status"})
)

func SetupMetrics(db *gorm.DB) error {
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "infra",
		Name:      "users",
		Help:      "Number of users managed by Infra.",
	}, func() float64 {
		count, err := data.Count(db, &models.User{}, &models.User{})
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
		count, err := data.Count(db, &models.Group{}, &models.Group{})
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
		count, err := data.Count(db, &models.Grant{}, &models.Grant{})
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
		count, err := data.Count(db, &models.Provider{}, &models.Provider{})
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
		count, err := data.Count(db, &models.Destination{}, &models.Destination{})
		if err != nil {
			logging.S.Warnf("destinations: %w", err)
			return 0
		}

		return float64(*count)
	})

	return nil
}
