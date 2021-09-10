package metrics

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/status"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RecordAPICallDuration produces a measurement for the APICallDuration view.
func RecordAPICallDuration(ctx context.Context, operation, status string, gvk schema.GroupVersionKind, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind), tag.Upsert(KeyStatus, status))
	measurement := APICallDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordReconcilerErrors produces a measurement for the ReconcilerErrors view.
func RecordReconcilerErrors(ctx context.Context, component string, numErrors int) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyComponent, component))
	measurement := ReconcilerErrors.M(int64(numErrors))
	stats.Record(tagCtx, measurement)
}

// RecordReconcilerNonBlockingErrors produces a measurement for the ReconcilerNonBlockingErrors view.
func RecordReconcilerNonBlockingErrors(ctx context.Context, errs status.MultiError) {
	errMeasurement := ReconcilerNonBlockingErrors.M(1)
	for _, err := range status.ToCSE(errs) {
		tagContext, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyErrorCode, err.Code))
		stats.Record(tagContext, errMeasurement)
	}
}

// RecordRenderingErrors produces a measurement for the RenderingErrors view.
func RecordRenderingErrors(ctx context.Context, component string, numErrors int, errCode string) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyComponent, component), tag.Upsert(KeyErrorCode, errCode))
	measurement := RenderingErrors.M(int64(numErrors))
	stats.Record(tagCtx, measurement)
}

// RecordReconcileDuration produces a measurement for the ReconcileDuration view.
func RecordReconcileDuration(ctx context.Context, status string, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyStatus, status))
	measurement := ReconcileDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordParserDuration produces a measurement for the ParserDuration view.
func RecordParserDuration(ctx context.Context, trigger, source, status string, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyStatus, status), tag.Upsert(KeyTrigger, trigger), tag.Upsert(KeyParserSource, source))
	measurement := ParserDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordLastSync produces a measurement for the LastSync view.
func RecordLastSync(ctx context.Context, commit string, timestamp time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyCommit, commit))
	measurement := LastSync.M(timestamp.Unix())
	stats.Record(tagCtx, measurement)
}

// RecordParseErrorAndDuration produces measurements for the ParseDuration and ParseErrors views.
func RecordParseErrorAndDuration(ctx context.Context, errs status.MultiError, startTime time.Time) {
	durationTagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyStatus, StatusTagKey(errs)))
	durationMeasurement := ParseDuration.M(time.Since(startTime).Seconds())
	stats.Record(durationTagCtx, durationMeasurement)

	errsMeasurement := ParseErrors.M(1)
	for _, err := range status.ToCSE(errs) {
		tagContext, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyErrorCode, err.Code))
		stats.Record(tagContext, errsMeasurement)
	}
}

// RecordDeclaredResources produces a measurement for the DeclaredResources view.
func RecordDeclaredResources(ctx context.Context, numResources int) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()))
	measurement := DeclaredResources.M(int64(numResources))
	stats.Record(tagCtx, measurement)
}

// RecordApplyOperation produces a measurement for the ApplyOperations view.
func RecordApplyOperation(ctx context.Context, operation, status string, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind), tag.Upsert(KeyStatus, status))
	measurement := ApplyOperations.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordLastApplyAndDuration produces measurements for the ApplyDuration and LastApplyTimestamp views.
func RecordLastApplyAndDuration(ctx context.Context, status, commit string, startTime time.Time) {
	now := time.Now()
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyStatus, status), tag.Upsert(KeyCommit, commit))

	durationMeasurement := ApplyDuration.M(now.Sub(startTime).Seconds())
	lastApplyMeasurement := LastApply.M(now.Unix())

	stats.Record(tagCtx, durationMeasurement, lastApplyMeasurement)
}

// RecordResourceFight produces measurements for the ResourceFights view.
func RecordResourceFight(ctx context.Context, operation string, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind))
	measurement := ResourceFights.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordWatches produces measurements for the Watches view.
func RecordWatches(ctx context.Context, gvk schema.GroupVersionKind, count int) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyType, gvk.Kind))
	measurement := Watches.M(int64(count))
	stats.Record(tagCtx, measurement)
}

// RecordWatchManagerUpdatesDuration produces measurements for the WatchManagerUpdatesDuration view.
func RecordWatchManagerUpdatesDuration(ctx context.Context, status string, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyStatus, status))
	measurement := WatchManagerUpdatesDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordRemediateDuration produces measurements for the RemediateDuration view.
func RecordRemediateDuration(ctx context.Context, status string, gvk schema.GroupVersionKind, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyStatus, status), tag.Upsert(KeyType, gvk.Kind))
	measurement := RemediateDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordResourceConflict produces measurements for the ResourceConflicts view.
func RecordResourceConflict(ctx context.Context, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyType, gvk.Kind))
	measurement := ResourceConflicts.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordInternalError produces measurements for the InternalErrors view.
func RecordInternalError(ctx context.Context, source string) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()), tag.Upsert(KeyInternalErrorSource, source))
	measurement := InternalErrors.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordRenderingCount produces measurements for the RenderingCount view.
func RecordRenderingCount(ctx context.Context) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()))
	measurement := RenderingCount.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordSkipRenderingCount produces measurements for the SkipRenderingCount view.
func RecordSkipRenderingCount(ctx context.Context) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconciler, ReconcilerTagKey()))
	measurement := SkipRenderingCount.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordResourceOverrideCount produces measurements for the ResourceOverrideCount view.
func RecordResourceOverrideCount(ctx context.Context, reconcilerType, containerName, resourceType string) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyReconcilerType, reconcilerType), tag.Upsert(KeyContainer, containerName), tag.Upsert(KeyContainer, containerName))
	measurement := ResourceOverrideCount.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordGitSyncDepthOverrideCount produces measurements for the GitSyncDepthOverrideCount view.
func RecordGitSyncDepthOverrideCount(ctx context.Context) {
	measurement := GitSyncDepthOverrideCount.M(1)
	stats.Record(ctx, measurement)
}

// RecordNoSSLVerifyCount produces measurements for the NoSSLVerifyCount view.
func RecordNoSSLVerifyCount(ctx context.Context) {
	measurement := NoSSLVerifyCount.M(1)
	stats.Record(ctx, measurement)
}
