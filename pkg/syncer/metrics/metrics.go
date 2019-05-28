package metrics

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics
var (
	APICallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of durations of API server calls",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "api_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1},
		},
		// operation: create, patch, update, delete
		// type: resource kind
		// status: success, error
		[]string{"operation", "type", "status"},
	)
	ControllerRestarts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total restart count for the NamespaceConfig and ClusterConfig controllers",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "controller_restarts_total",
		},
		// source: sync, crd, retry
		[]string{"source"},
	)
	Operations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Help:      "Total operations that have been performed to sync resources to source of truth",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "operations_total",
		},
		// operation: create, update, delete
		// type: resource kind
		// status: success, error
		[]string{"operation", "type", "status"},
	)
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Help:      "Distribution of syncer reconciliation durations",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "reconcile_duration_seconds",
			Buckets:   []float64{.001, .01, .1, 1, 10, 100},
		},
		// type: cluster, crd, namespace, repo, sync
		// status: success, error
		[]string{"type", "status"},
	)
	ReconcileEventTimes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help:      "Timestamps when syncer reconcile events occurred",
			Namespace: configmanagement.MetricsNamespace,
			Subsystem: "syncer",
			Name:      "reconcile_event_timestamps",
		},
		// type: cluster, crd, namespace, repo, sync
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		APICallDuration,
		ControllerRestarts,
		Operations,
		ReconcileDuration,
		ReconcileEventTimes,
	)
}

// StatusLabel returns a string representation of the given error appropriate for the status label
// of a Prometheus metric.
func StatusLabel(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}
