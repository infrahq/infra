package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
)

type metricValue struct {
	Value       float64
	LabelValues []string
}

// collector implements the prometheus.Collector interface
type collector struct {
	desc        *prometheus.Desc
	valueType   prometheus.ValueType
	collectFunc func() []metricValue
}

func newCollector(opts prometheus.Opts, valueType prometheus.ValueType, variableLabels []string, collectFunc func() []metricValue) *collector {
	fqname := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)
	return &collector{
		desc:        prometheus.NewDesc(fqname, opts.Help, variableLabels, opts.ConstLabels),
		valueType:   valueType,
		collectFunc: collectFunc,
	}
}

// NewGaugeCollector creates a collect with type Gauge
func NewGaugeCollector(opts prometheus.Opts, variableLabels []string, collectFunc func() []metricValue) *collector {
	return newCollector(opts, prometheus.GaugeValue, variableLabels, collectFunc)
}

// Describe is implemented by DescribeByCollect
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect implements Collector. It create a set of constant metrics with the values and labels
// as described by collectFunc
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	for _, metricValue := range c.collectFunc() {
		ch <- prometheus.MustNewConstMetric(c.desc, c.valueType, float64(metricValue.Value), metricValue.LabelValues...)
	}
}

func setupMetrics(db *gorm.DB) *prometheus.Registry {
	registry := prometheus.NewRegistry()

	if rawDB, err := db.DB(); err == nil {
		registry.MustRegister(collectors.NewDBStatsCollector(rawDB, db.Dialector.Name()))
	}

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

	registry.MustRegister(NewGaugeCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "users",
		Help:      "The total number of users",
	}, []string{}, func() []metricValue {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM identities").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("users")
			return []metricValue{}
		}

		values := make([]metricValue, 0, len(results))
		for _, result := range results {
			values = append(values, metricValue{Value: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(NewGaugeCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "groups",
		Help:      "The total number of groups",
	}, []string{}, func() []metricValue {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM groups").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("groups")
			return []metricValue{}
		}

		values := make([]metricValue, 0, len(results))
		for _, result := range results {
			values = append(values, metricValue{Value: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(NewGaugeCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "grants",
		Help:      "The total number of grants",
	}, []string{}, func() []metricValue {
		var results []struct {
			Count int
		}

		if err := db.Raw("SELECT COUNT(*) as count FROM grants").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("grants")
			return []metricValue{}
		}

		values := make([]metricValue, 0, len(results))
		for _, result := range results {
			values = append(values, metricValue{Value: float64(result.Count), LabelValues: []string{}})
		}

		return values
	}))

	registry.MustRegister(NewGaugeCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "providers",
		Help:      "The total number of providers",
	}, []string{"kind"}, func() []metricValue {
		var results []struct {
			Kind  string
			Count int
		}

		if err := db.Raw("SELECT kind, COUNT(*) as count FROM providers GROUP BY kind").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("providers")
			return []metricValue{}
		}

		values := make([]metricValue, 0, len(results))
		for _, result := range results {
			values = append(values, metricValue{Value: float64(result.Count), LabelValues: []string{result.Kind}})
		}

		return values
	}))

	registry.MustRegister(NewGaugeCollector(prometheus.Opts{
		Namespace: "infra",
		Name:      "destinations",
		Help:      "The total number of destinations",
	}, []string{"version"}, func() []metricValue {
		var results []struct {
			Version string
			Count   int
		}

		if err := db.Raw("SELECT version, COUNT(*) as count FROM destinations GROUP BY version").Scan(&results).Error; err != nil {
			logging.L.Warn().Err(err).Msg("destinations")
			return []metricValue{}
		}

		values := make([]metricValue, 0, len(results))
		for _, result := range results {
			values = append(values, metricValue{Value: float64(result.Count), LabelValues: []string{result.Version}})
		}

		return values
	}))

	return registry
}
