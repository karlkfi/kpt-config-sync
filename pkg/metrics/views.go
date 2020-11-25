package metrics

import (
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var distributionBounds = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

var (
	apiCallDurationView = &view.View{
		Name:        APICallDuration.Name(),
		Measure:     APICallDuration,
		Description: "The latency distribution of API server calls",
		TagKeys:     []tag.Key{KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	reconcilerErrorsView = &view.View{
		Name:        ReconcilerErrors.Name(),
		Measure:     ReconcilerErrors,
		Description: "The current number of errors in the RootSync and RepoSync reconcilers",
		TagKeys:     []tag.Key{KeyScope, KeyComponent},
		Aggregation: view.LastValue(),
	}

	reconcileDurationView = &view.View{
		Name:        ReconcileDuration.Name(),
		Measure:     ReconcileDuration,
		Description: "The latency distribution of RootSync and RepoSync reconcile events",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	lastSyncTimestampView = &view.View{
		Name:        LastSync.Name(),
		Measure:     LastSync,
		Description: "The timestamp of the most recent sync from Git grouped by the scope",
		TagKeys:     []tag.Key{KeyScope},
		Aggregation: view.LastValue(),
	}

	parseDurationView = &view.View{
		Name:        ParseDuration.Name(),
		Measure:     ParseDuration,
		Description: "The latency distribution of parse events",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	parseErrorsView = &view.View{
		Name:        ParseErrors.Name() + "_total",
		Measure:     ParseErrors,
		Description: "The total number of errors that occurred during parsing",
		TagKeys:     []tag.Key{KeyScope, KeyErrorCode},
		Aggregation: view.Count(),
	}

	declaredResourcesView = &view.View{
		Name:        DeclaredResources.Name(),
		Measure:     DeclaredResources,
		Description: "The current number of declared resources parsed from Git",
		TagKeys:     []tag.Key{KeyScope},
		Aggregation: view.LastValue(),
	}

	applyOperationsView = &view.View{
		Name:        ApplyOperations.Name() + "_total",
		Measure:     ApplyOperations,
		Description: "The total number of operations that have been performed to sync resources to source of truth",
		TagKeys:     []tag.Key{KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Count(),
	}

	applyDurationView = &view.View{
		Name:        ApplyDuration.Name(),
		Measure:     ApplyDuration,
		Description: "The latency distribution of applier resource sync events",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	lastApplyTimestampView = &view.View{
		Name:        LastApply.Name(),
		Measure:     LastApply,
		Description: "The timestamp of the most recent applier resource sync event",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.LastValue(),
	}

	resourceFightsView = &view.View{
		Name:        ResourceFights.Name() + "_total",
		Measure:     ResourceFights,
		Description: "The total number of resources that are being synced too frequently",
		TagKeys:     []tag.Key{KeyScope, KeyType},
		Aggregation: view.Count(),
	}

	watchesView = &view.View{
		Name:        Watches.Name(),
		Measure:     Watches,
		Description: "The current number of watches on the declared resources",
		TagKeys:     []tag.Key{KeyScope, KeyType},
		Aggregation: view.Sum(),
	}

	watchManagerUpdatesView = &view.View{
		Name:        WatchManagerUpdates.Name() + "_total",
		Measure:     WatchManagerUpdates,
		Description: "The total number of times the watch manager updates the watches on the declared resources",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.Count(),
	}

	watchManagerUpdatesDurationView = &view.View{
		Name:        WatchManagerUpdatesDuration.Name(),
		Measure:     WatchManagerUpdatesDuration,
		Description: "The latency distribution of watch manager updates",
		TagKeys:     []tag.Key{KeyScope, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	remediateDurationView = &view.View{
		Name:        RemediateDuration.Name(),
		Measure:     RemediateDuration,
		Description: "The latency distribution of remediator reconciliation events",
		TagKeys:     []tag.Key{KeyStatus, KeyType},
		Aggregation: view.Distribution(distributionBounds...),
	}

	resourceConflictsView = &view.View{
		Name:        ResourceConflicts.Name() + "_total",
		Measure:     ResourceConflicts,
		Description: "The total number of resource conflicts resulting from a mismatch between the cached resources and cluster resources",
		TagKeys:     []tag.Key{KeyType},
		Aggregation: view.Count(),
	}

	internalErrorsView = &view.View{
		Name:        InternalErrors.Name() + "_total",
		Measure:     InternalErrors,
		Description: "The total number of internal errors triggered by Config Sync",
		TagKeys:     []tag.Key{KeySource},
		Aggregation: view.Count(),
	}
)
