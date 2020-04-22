package state

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the Prometheus metrics for the monitor state.
var metrics = struct {
	Configs     *prometheus.GaugeVec
	Errors      *prometheus.GaugeVec
	LastImport  prometheus.Gauge
	LastSync    prometheus.Gauge
	SyncLatency prometheus.Histogram
}{
	prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Current number of configs (cluster and namespace) grouped by their sync status",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "configs",
		},
		// status: synced, stale, error
		[]string{"status"},
	),
	prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Current number of errors in the config repo, grouped by the component where they occurred",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "errors",
		},
		// component: source, importer, syncer
		[]string{"component"},
	),
	prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Timestamp of the most recent import",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "last_import_timestamp",
		},
	),
	prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Timestamp of the most recent sync",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "last_sync_timestamp",
		},
	),
	prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Help:      "Distribution of the latencies between importing and syncing each config",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "sync_latency_seconds",
			Buckets:   prometheus.DefBuckets,
		},
	),
}

func init() {
	prometheus.MustRegister(
		metrics.Configs,
		metrics.Errors,
		metrics.LastImport,
		metrics.LastSync,
		metrics.SyncLatency,
	)
}
