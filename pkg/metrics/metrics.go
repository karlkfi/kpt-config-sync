package metrics

import "go.opencensus.io/stats"

var (
	// APICallDuration metric measures the latency of API server calls.
	APICallDuration = stats.Float64(
		"api_duration_seconds",
		"The duration of API server calls in seconds",
		stats.UnitSeconds)

	// ReconcilerErrors metric measures the number of errors in the reconciler.
	ReconcilerErrors = stats.Int64(
		"reconciler_errors",
		"The number of errors in the reconciler",
		stats.UnitDimensionless)

	// ReconcilerNonBlockingErrors metric measures the number of non-blocking errors in the reconciler.
	ReconcilerNonBlockingErrors = stats.Int64(
		"reconciler_non_blocking_errors",
		"The number of non-blocking errors in the reconciler",
		stats.UnitDimensionless)

	// RenderingErrors metric measures the number of errors in the rendering process.
	RenderingErrors = stats.Int64(
		"rendering_errors",
		"The number of errors in the rendering process",
		stats.UnitDimensionless)

	// PipelineError metric measures the error by components when syncing a commit
	PipelineError = stats.Int64(
		"pipeline_error_observed",
		"A boolean indicates if any error happens from different stages when syncing a commit",
		stats.UnitDimensionless)

	// ReconcileDuration metric measures the latency of reconcile events.
	ReconcileDuration = stats.Float64(
		"reconcile_duration_seconds",
		"The duration of reconcile events in seconds",
		stats.UnitSeconds)

	// ParserDuration metric measures the latency of the parse-apply-watch loop.
	ParserDuration = stats.Float64(
		"parser_duration_seconds",
		"The duration of the parse-apply-watch loop in seconds",
		stats.UnitSeconds)

	// LastSync metric measures the timestamp of the latest Git sync.
	LastSync = stats.Int64(
		"last_sync_timestamp",
		"The timestamp of the most recent sync from Git",
		stats.UnitDimensionless)

	// ParseDuration metric measures the latency of parse events.
	ParseDuration = stats.Float64(
		"parse_duration_seconds",
		"The duration of parse events in seconds",
		stats.UnitSeconds)

	// ParseErrors metric measures the number of parse errors.
	ParseErrors = stats.Int64(
		"parse_errors",
		"The number of errors that occurred during parsing",
		stats.UnitDimensionless)

	// DeclaredResources metric measures the number of declared resources parsed from Git.
	DeclaredResources = stats.Int64(
		"declared_resources",
		"The number of declared resources parsed from Git",
		stats.UnitDimensionless)

	// ApplyOperations metric measures the number of applier apply events.
	ApplyOperations = stats.Int64(
		"apply_operations",
		"The number of operations that have been performed to sync resources to source of truth",
		stats.UnitDimensionless)

	// ApplyDuration metric measures the latency of applier apply events.
	ApplyDuration = stats.Float64(
		"apply_duration_seconds",
		"The duration of applier events in seconds",
		stats.UnitSeconds)

	// LastApply metric measures the timestamp of the most recent applier apply event.
	LastApply = stats.Int64(
		"last_apply_timestamp",
		"The timestamp of the most recent applier event",
		stats.UnitDimensionless)

	// ResourceFights metric measures the number of resource fights.
	ResourceFights = stats.Int64(
		"resource_fights",
		"The number of resources that are being synced too frequently",
		stats.UnitDimensionless)

	// Watches metric measures the number of watches on the declared resources.
	Watches = stats.Int64(
		"watches",
		"The number of watches on the declared resources",
		stats.UnitDimensionless)

	// WatchManagerUpdatesDuration metric measures the latency of watch manager updates.
	WatchManagerUpdatesDuration = stats.Float64(
		"watch_manager_updates_duration_seconds",
		"The duration of watch manager updates",
		stats.UnitSeconds)

	// RemediateDuration metric measures the latency of remediator reconciliation events.
	RemediateDuration = stats.Float64(
		"remediate_duration_seconds",
		"The duration of remediator reconciliation events",
		stats.UnitSeconds)

	// ResourceConflicts metric measures the number of resource conflicts.
	ResourceConflicts = stats.Int64(
		"resource_conflicts",
		"The number of resource conflicts resulting from a mismatch between the cached resources and cluster resources",
		stats.UnitDimensionless)

	// InternalErrors metric measures the number of unexpected internal errors triggered by defensive checks in Config Sync.
	InternalErrors = stats.Int64(
		"internal_errors",
		"The number of internal errors triggered by Config Sync",
		stats.UnitDimensionless)

	// RenderingCount metrics measures the number of renderings are performed.
	RenderingCount = stats.Int64(
		"rendering_count",
		"The number of renderings that are performed",
		stats.UnitDimensionless)

	// SkipRenderingCount metrics measures the number of renderings are skipped.
	SkipRenderingCount = stats.Int64(
		"skip_rendering_count",
		"The number of renderings that are skipped",
		stats.UnitDimensionless)

	// ResourceOverrideCount metric measures the number of RootSync/RepoSync objects including the `spec.override.resources` field.
	ResourceOverrideCount = stats.Int64(
		"resource_override_count",
		"The number of RootSync/RepoSync objects including the `spec.override.resources` field",
		stats.UnitDimensionless)

	// GitSyncDepthOverrideCount metric measures the number of RootSync/RepoSync objects including the `spec.override.gitSyncDepth` field.
	GitSyncDepthOverrideCount = stats.Int64(
		"git_sync_depth_override_count",
		"The number of RootSync/RepoSync objects including the `spec.override.gitSyncDepth` field",
		stats.UnitDimensionless)

	// NoSSLVerifyCount metric measures the number of RootSync/RepoSync objects whose `spec.git.noSSLVerify` field is set to `true`.
	NoSSLVerifyCount = stats.Int64(
		"no_ssl_verify_count",
		"The number of RootSync/RepoSync objects whose `spec.git.noSSLVerify` field is set to `true`",
		stats.UnitDimensionless)
)
