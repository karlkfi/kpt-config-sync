package metrics

import (
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/pkg/errors"
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

// RegisterStackdriverExporter creates and starts the OpenCensus Stackdriver metrics exporter.
func RegisterStackdriverExporter() (*stackdriver.Exporter, error) {
	sde, err := stackdriver.NewExporter(stackdriver.Options{
		MetricPrefix: namespace,
		// ReportingInterval sets the frequency of reporting metrics to Stackdriver backend.
		ReportingInterval: 60 * time.Second,
	})
	if err != nil {
		return nil, errors.Errorf("failed to create Stackdriver exporter: %v", err)
	}

	// Start the metrics exporter
	if err := sde.StartMetricsExporter(); err != nil {
		return nil, errors.Errorf("failed to start Stackdriver exporter: %v", err)
	}
	return sde, nil
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
