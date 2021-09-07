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
		TagKeys:     []tag.Key{KeyReconciler, KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ReconcilerErrorsView aggregates the ReconcilerErrors metric measurements.
	ReconcilerErrorsView = &view.View{
		Name:        ReconcilerErrors.Name(),
		Measure:     ReconcilerErrors,
		Description: "The current number of errors in the RootSync and RepoSync reconcilers",
		TagKeys:     []tag.Key{KeyReconciler, KeyComponent},
		Aggregation: view.LastValue(),
	}

	// ReconcilerNonBlockingErrorsView aggregates the ReconcilerNonBlockingErrors metric measurements.
	ReconcilerNonBlockingErrorsView = &view.View{
		Name:        ReconcilerNonBlockingErrors.Name(),
		Measure:     ReconcilerNonBlockingErrors,
		Description: "The current number of non-blocking errors in the RootSync and RepoSync reconcilers",
		TagKeys:     []tag.Key{KeyReconciler, KeyErrorCode},
		Aggregation: view.LastValue(),
	}

	// RenderingErrorsView aggregates the RenderingErrors metric measurements.
	RenderingErrorsView = &view.View{
		Name:        RenderingErrors.Name(),
		Measure:     RenderingErrors,
		Description: "The current number of errors in the RootSync and RepoSync rendering process",
		TagKeys:     []tag.Key{KeyReconciler, KeyComponent},
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

	// ParserDurationView aggregates the ParserDuration metric measurements.
	ParserDurationView = &view.View{
		Name:        ParserDuration.Name(),
		Measure:     ParserDuration,
		Description: "The latency distribution of the parse-apply-watch loop",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus, KeyTrigger, KeyParserSource},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// LastSyncTimestampView aggregates the LastSyncTimestamp metric measurements.
	LastSyncTimestampView = &view.View{
		Name:        LastSync.Name(),
		Measure:     LastSync,
		Description: "The timestamp of the most recent sync from Git",
		TagKeys:     []tag.Key{KeyReconciler, KeyCommit},
		Aggregation: view.LastValue(),
	}

	// ParseDurationView aggregates the ParseDuration metric measurements.
	ParseDurationView = &view.View{
		Name:        ParseDuration.Name(),
		Measure:     ParseDuration,
		Description: "The latency distribution of parse events",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ParseErrorsView aggregates the ParseErrors metric measurements.
	ParseErrorsView = &view.View{
		Name:        ParseErrors.Name() + "_total",
		Measure:     ParseErrors,
		Description: "The total number of errors that occurred during parsing",
		TagKeys:     []tag.Key{KeyReconciler, KeyErrorCode},
		Aggregation: view.Count(),
	}

	// DeclaredResourcesView aggregates the DeclaredResources metric measurements.
	DeclaredResourcesView = &view.View{
		Name:        DeclaredResources.Name(),
		Measure:     DeclaredResources,
		Description: "The current number of declared resources parsed from Git",
		TagKeys:     []tag.Key{KeyReconciler},
		Aggregation: view.LastValue(),
	}

	// ApplyOperationsView aggregates the ApplyOps metric measurements.
	ApplyOperationsView = &view.View{
		Name:        ApplyOperations.Name() + "_total",
		Measure:     ApplyOperations,
		Description: "The total number of operations that have been performed to sync resources to source of truth",
		TagKeys:     []tag.Key{KeyReconciler, KeyOperation, KeyType, KeyStatus},
		Aggregation: view.Count(),
	}

	// ApplyDurationView aggregates the ApplyDuration metric measurements.
	ApplyDurationView = &view.View{
		Name:        ApplyDuration.Name(),
		Measure:     ApplyDuration,
		Description: "The latency distribution of applier resource sync events",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// LastApplyTimestampView aggregates the LastApplyTimestamp metric measurements.
	LastApplyTimestampView = &view.View{
		Name:        LastApply.Name(),
		Measure:     LastApply,
		Description: "The timestamp of the most recent applier resource sync event",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus, KeyCommit},
		Aggregation: view.LastValue(),
	}

	// ResourceFightsView aggregates the ResourceFights metric measurements.
	ResourceFightsView = &view.View{
		Name:        ResourceFights.Name() + "_total",
		Measure:     ResourceFights,
		Description: "The total number of resources that are being synced too frequently",
		TagKeys:     []tag.Key{KeyReconciler, KeyOperation, KeyType},
		Aggregation: view.Count(),
	}

	// WatchesView aggregates the Watches metric measurements.
	WatchesView = &view.View{
		Name:        Watches.Name(),
		Measure:     Watches,
		Description: "The current number of watches on the declared resources",
		TagKeys:     []tag.Key{KeyReconciler, KeyType},
		Aggregation: view.Sum(),
	}

	// WatchManagerUpdatesDurationView aggregates the WatchManagerUpdatesDuration metric measurements.
	WatchManagerUpdatesDurationView = &view.View{
		Name:        WatchManagerUpdatesDuration.Name(),
		Measure:     WatchManagerUpdatesDuration,
		Description: "The latency distribution of watch manager updates",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// RemediateDurationView aggregates the RemediateDuration metric measurements.
	RemediateDurationView = &view.View{
		Name:        RemediateDuration.Name(),
		Measure:     RemediateDuration,
		Description: "The latency distribution of remediator reconciliation events",
		TagKeys:     []tag.Key{KeyReconciler, KeyStatus, KeyType},
		Aggregation: view.Distribution(distributionBounds...),
	}

	// ResourceConflictsView aggregates the ResourceConflicts metric measurements.
	ResourceConflictsView = &view.View{
		Name:        ResourceConflicts.Name() + "_total",
		Measure:     ResourceConflicts,
		Description: "The total number of resource conflicts resulting from a mismatch between the cached resources and cluster resources",
		TagKeys:     []tag.Key{KeyReconciler, KeyType},
		Aggregation: view.Count(),
	}

	// InternalErrorsView aggregates the InternalErrors metric measurements.
	InternalErrorsView = &view.View{
		Name:        InternalErrors.Name() + "_total",
		Measure:     InternalErrors,
		Description: "The total number of internal errors triggered by Config Sync",
		TagKeys:     []tag.Key{KeyReconciler, KeyInternalErrorSource},
		Aggregation: view.Count(),
	}

	// RenderingCountView aggregates the RenderingCount metric measurements.
	RenderingCountView = &view.View{
		Name:        RenderingCount.Name() + "_total",
		Measure:     RenderingCount,
		Description: "The total number of renderings that are skipped",
		TagKeys:     []tag.Key{KeyReconciler},
		Aggregation: view.Count(),
	}

	// SkipRenderingCountView aggregates the SkipRenderingCount metric measurements.
	SkipRenderingCountView = &view.View{
		Name:        SkipRenderingCount.Name() + "_total",
		Measure:     SkipRenderingCount,
		Description: "The total number of renderings that are performed",
		TagKeys:     []tag.Key{KeyReconciler},
		Aggregation: view.Count(),
	}
)
