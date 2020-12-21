package metrics

import (
	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
)

var (
	// The namespace for the OpenCensus Prometheus and Stackdriver metrics.
	namespace = "configsync"
)

// RegisterPrometheusExporter creates the OpenCensus Prometheus metrics exporter.
func RegisterPrometheusExporter() (*prometheus.Exporter, error) {
	return prometheus.NewExporter(prometheus.Options{
		Namespace: namespace,
	})
}

// RegisterReconcilerManagerMetricsViews registers the views so that recorded metrics can be exported in the reconciler manager.
func RegisterReconcilerManagerMetricsViews() error {
	return view.Register(reconcileDurationView)
}

// RegisterReconcilerMetricsViews registers the views so that recorded metrics can be exported in the reconcilers.
func RegisterReconcilerMetricsViews() error {
	return view.Register(
		apiCallDurationView,
		reconcilerErrorsView,
		lastSyncTimestampView,
		parseDurationView,
		parseErrorsView,
		declaredResourcesView,
		applyOperationsView,
		applyDurationView,
		lastApplyTimestampView,
		resourceFightsView,
		watchesView,
		watchManagerUpdatesView,
		watchManagerUpdatesDurationView,
		remediateDurationView,
		resourceConflictsView,
		internalErrorsView)
}
