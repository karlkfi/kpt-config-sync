package metrics

import (
	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
)

// RegisterOCAgentExporter creates the OC Agent metrics exporter.
func RegisterOCAgentExporter() (*ocagent.Exporter, error) {
	oce, err := ocagent.NewExporter(
		ocagent.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	view.RegisterExporter(oce)
	return oce, nil
}

// RegisterReconcilerManagerMetricsViews registers the views so that recorded metrics can be exported in the reconciler manager.
func RegisterReconcilerManagerMetricsViews() error {
	return view.Register(ReconcileDurationView)
}

// RegisterReconcilerMetricsViews registers the views so that recorded metrics can be exported in the reconcilers.
func RegisterReconcilerMetricsViews() error {
	return view.Register(
		APICallDurationView,
		ReconcilerErrorsView,
		RenderingErrorsView,
		ParserDurationView,
		LastSyncTimestampView,
		ParseDurationView,
		ParseErrorsView,
		DeclaredResourcesView,
		ApplyOperationsView,
		ApplyDurationView,
		LastApplyTimestampView,
		ResourceFightsView,
		WatchesView,
		WatchManagerUpdatesDurationView,
		RemediateDurationView,
		ResourceConflictsView,
		InternalErrorsView,
		RenderingCountView,
		SkipRenderingCountView)
}
