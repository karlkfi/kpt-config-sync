package importer

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the Prometheus metrics for the Importer.
var Metrics = struct {
	CycleDuration    *prometheus.HistogramVec
	NamespaceConfigs prometheus.Gauge
}{
	CycleDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of durations of cycles that the importer has attempted to complete",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "cycle_duration_seconds",
		},
		// status: success, error
		[]string{"status"},
	),
	NamespaceConfigs: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Number of namespace configs present in current state",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "namespace_configs",
		},
	),
}

func init() {
	prometheus.MustRegister(
		Metrics.CycleDuration,
		Metrics.NamespaceConfigs,
	)
}
