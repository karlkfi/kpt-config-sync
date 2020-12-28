package metrics

import (
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var distributionBounds = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

var (
	// APICallDurationView aggregates the APICallDuration metric measurements.
	APICallDurationView = &view.View{
		Name:        APICallDuration.Name(),
		Measure:     APICallDuration,
		Description: "The latency distribution of API server calls",
		TagKeys:     []tag.Key{KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ReconcilerErrorsView aggregates the ReconcilerErrors metric measurements.
	ReconcilerErrorsView = &view.View{
		Name:        ReconcilerErrors.Name(),
		Measure:     ReconcilerErrors,
		Description: "The current number of errors in the RootSync and RepoSync reconcilers",
		TagKeys:     []tag.Key{KeyComponent},
		Aggregation: view.LastValue(),
	}

	// ReconcileDurationView aggregates the ReconcileDuration metric measurements.
	ReconcileDurationView = &view.View{
		Name:        ReconcileDuration.Name(),
		Measure:     ReconcileDuration,
		Description: "The latency distribution of RootSync and RepoSync reconcile events",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// LastSyncTimestampView aggregates the LastSyncTimestamp metric measurements.
	LastSyncTimestampView = &view.View{
		Name:        LastSync.Name(),
		Measure:     LastSync,
		Description: "The timestamp of the most recent sync from Git grouped by the scope",
		Aggregation: view.LastValue(),
	}

	// ParseDurationView aggregates the ParseDuration metric measurements.
	ParseDurationView = &view.View{
		Name:        ParseDuration.Name(),
		Measure:     ParseDuration,
		Description: "The latency distribution of parse events",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ParseErrorsView aggregates the ParseErrors metric measurements.
	ParseErrorsView = &view.View{
		Name:        ParseErrors.Name() + "_total",
		Measure:     ParseErrors,
		Description: "The total number of errors that occurred during parsing",
		TagKeys:     []tag.Key{KeyErrorCode},
		Aggregation: view.Count(),
	}

	// DeclaredResourcesView aggregates the DeclaredResources metric measurements.
	DeclaredResourcesView = &view.View{
		Name:        DeclaredResources.Name(),
		Measure:     DeclaredResources,
		Description: "The current number of declared resources parsed from Git",
		Aggregation: view.LastValue(),
	}

	// ApplyOperationsView aggregates the ApplyOperations metric measurements.
	ApplyOperationsView = &view.View{
		Name:        ApplyOperations.Name() + "_total",
		Measure:     ApplyOperations,
		Description: "The total number of operations that have been performed to sync resources to source of truth",
		TagKeys:     []tag.Key{KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Count(),
	}

	// ApplyDurationView aggregates the ApplyDuration metric measurements.
	ApplyDurationView = &view.View{
		Name:        ApplyDuration.Name(),
		Measure:     ApplyDuration,
		Description: "The latency distribution of applier resource sync events",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// LastApplyTimestampView aggregates the LastApplyTimestamp metric measurements.
	LastApplyTimestampView = &view.View{
		Name:        LastApply.Name(),
		Measure:     LastApply,
		Description: "The timestamp of the most recent applier resource sync event",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.LastValue(),
	}

	// ResourceFightsView aggregates the ResourceFights metric measurements.
	ResourceFightsView = &view.View{
		Name:        ResourceFights.Name() + "_total",
		Measure:     ResourceFights,
		Description: "The total number of resources that are being synced too frequently",
		TagKeys:     []tag.Key{KeyOperation, KeyType},
		Aggregation: view.Count(),
	}

	// WatchesView aggregates the Watches metric measurements.
	WatchesView = &view.View{
		Name:        Watches.Name(),
		Measure:     Watches,
		Description: "The current number of watches on the declared resources",
		TagKeys:     []tag.Key{KeyType},
		Aggregation: view.Sum(),
	}

	// WatchManagerUpdatesView aggregates the WatchManagerUpdates metric measurements.
	WatchManagerUpdatesView = &view.View{
		Name:        WatchManagerUpdates.Name() + "_total",
		Measure:     WatchManagerUpdates,
		Description: "The total number of times the watch manager updates the watches on the declared resources",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.Count(),
	}

	// WatchManagerUpdatesDurationView aggregates the WatchManagerUpdatesDuration metric measurements.
	WatchManagerUpdatesDurationView = &view.View{
		Name:        WatchManagerUpdatesDuration.Name(),
		Measure:     WatchManagerUpdatesDuration,
		Description: "The latency distribution of watch manager updates",
		TagKeys:     []tag.Key{KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// RemediateDurationView aggregates the RemediateDuration metric measurements.
	RemediateDurationView = &view.View{
		Name:        RemediateDuration.Name(),
		Measure:     RemediateDuration,
		Description: "The latency distribution of remediator reconciliation events",
		TagKeys:     []tag.Key{KeyStatus, KeyType},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ResourceConflictsView aggregates the ResourceConflicts metric measurements.
	ResourceConflictsView = &view.View{
		Name:        ResourceConflicts.Name() + "_total",
		Measure:     ResourceConflicts,
		Description: "The total number of resource conflicts resulting from a mismatch between the cached resources and cluster resources",
		TagKeys:     []tag.Key{KeyType},
		Aggregation: view.Count(),
	}

	// InternalErrorsView aggregates the InternalErrors metric measurements.
	InternalErrorsView = &view.View{
		Name:        InternalErrors.Name() + "_total",
		Measure:     InternalErrors,
		Description: "The total number of internal errors triggered by Config Sync",
		TagKeys:     []tag.Key{KeySource},
		Aggregation: view.Count(),
	}
)
