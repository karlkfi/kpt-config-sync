package importer

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the Prometheus metrics for the Importer.
var Metrics = struct {
	APICallDuration  *prometheus.HistogramVec
	CycleDuration    *prometheus.HistogramVec
	NamespaceConfigs prometheus.Gauge
	Operations       *prometheus.CounterVec
}{
	APICallDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of durations of API server calls",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "api_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1},
		},
		// operation: create, update, delete
		// type: namespace, cluster, sync
		// status: success, error
		[]string{"operation", "type", "status"},
	),
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
	Operations: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total operations that have been performed to keep configs up-to-date with source of truth",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "importer",
			Name:      "operations_total",
		},
		// operation: create, update, delete
		// type: namespace, cluster, sync
		// status: success, error
		[]string{"operation", "type", "status"},
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
		Metrics.APICallDuration,
		Metrics.CycleDuration,
		Metrics.NamespaceConfigs,
		Metrics.Operations,
	)
}
