package state

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics contains the Prometheus metrics for the monitor state.
var Metrics = struct {
	ClusterNodes *prometheus.GaugeVec
	LastImport   prometheus.Gauge
	LastSync     prometheus.Gauge
	SyncLatency  prometheus.Histogram
}{
	prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Total number of policies (cluster and node) grouped by their sync status; should be similar to nomos_policy_importer_policy_nodes metric",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "policies",
		},
		[]string{"state"},
	),
	prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Timestamp of the most recent import",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "last_import_timestamp",
		},
	),
	prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Timestamp of the most recent sync",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "last_sync_timestamp",
		},
	),
	prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Help:      "Distribution of the latencies between importing and syncing each node",
			Namespace: policyhierarchy.MetricsNamespace,
			Subsystem: "monitor",
			Name:      "sync_latency_seconds",
			Buckets:   prometheus.DefBuckets,
		},
	),
}

func init() {
	prometheus.MustRegister(
		Metrics.ClusterNodes,
		Metrics.LastImport,
		Metrics.LastSync,
		Metrics.SyncLatency,
	)
}
