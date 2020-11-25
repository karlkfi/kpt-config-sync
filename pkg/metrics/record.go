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
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind), tag.Upsert(KeyStatus, status))
	measurement := APICallDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordReconcilerErrors produces a measurement for the ReconcilerErrors view.
func RecordReconcilerErrors(ctx context.Context, scope, component string, numErrors int) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyComponent, component))
	measurement := ReconcilerErrors.M(int64(numErrors))
	stats.Record(tagCtx, measurement)
}

// RecordReconcileDuration produces a measurement for the ReconcileDuration view.
func RecordReconcileDuration(ctx context.Context, scope string, status string, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyStatus, status))
	measurement := ReconcileDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordLastSync produces a measurement for the LastSync view.
func RecordLastSync(ctx context.Context, scope string, timestamp time.Time) {
	tagContext, _ := tag.New(ctx, tag.Upsert(KeyScope, scope))
	measurement := LastSync.M(timestamp.Unix())
	stats.Record(tagContext, measurement)
}

// RecordParseErrorsAndDuration produces measurements for the ParseDuration and ParseErrors views.
func RecordParseErrorsAndDuration(ctx context.Context, scope string, errs status.MultiError, startTime time.Time) {
	durationTagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyStatus, StatusTagKey(errs)))
	durationMeasurement := ParseDuration.M(time.Since(startTime).Seconds())
	stats.Record(durationTagCtx, durationMeasurement)

	errsMeasurement := ParseErrors.M(1)
	for _, err := range status.ToCSE(errs) {
		tagContext, _ := tag.New(context.Background(), tag.Upsert(KeyScope, scope), tag.Upsert(KeyErrorCode, err.Code))
		stats.Record(tagContext, errsMeasurement)
	}
}

// RecordDeclaredResources produces a measurement for the DeclaredResources view.
func RecordDeclaredResources(ctx context.Context, scope string, numResources int) {
	tagContext, _ := tag.New(ctx, tag.Upsert(KeyScope, scope))
	measurement := DeclaredResources.M(int64(numResources))
	stats.Record(tagContext, measurement)
}

// RecordApplyOperations produces a measurement for the ApplyOperations view.
func RecordApplyOperations(ctx context.Context, operation, status string, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind), tag.Upsert(KeyStatus, status))
	measurement := ApplyOperations.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordLastApplyAndDuration produces measurements for the ApplyDuration and LastApplyTimestamp views.
func RecordLastApplyAndDuration(ctx context.Context, scope string, status string, startTime time.Time) {
	now := time.Now()
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyStatus, status))

	durationMeasurement := ApplyDuration.M(now.Sub(startTime).Seconds())
	lastApplyMeasurement := LastApply.M(now.Unix())

	stats.Record(tagCtx, durationMeasurement, lastApplyMeasurement)
}

// RecordResourceFights produces measurements for the ResourceFights view.
func RecordResourceFights(ctx context.Context, operation string, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyOperation, operation), tag.Upsert(KeyType, gvk.Kind))
	measurement := ResourceFights.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordWatches produces measurements for the Watches view.
func RecordWatches(ctx context.Context, scope string, gvk schema.GroupVersionKind, count int) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyType, gvk.Kind))
	measurement := Watches.M(int64(count))
	stats.Record(tagCtx, measurement)
}

// RecordWatchManagerUpdatesAndDuration produces measurements for the WatchManagerUpdates and WatchManagerUpdatesDuration views.
func RecordWatchManagerUpdatesAndDuration(ctx context.Context, scope string, status string, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyScope, scope), tag.Upsert(KeyStatus, status))

	updatesMeasurement := WatchManagerUpdates.M(1)
	durationMeasurement := WatchManagerUpdatesDuration.M(time.Since(startTime).Seconds())

	stats.Record(tagCtx, updatesMeasurement, durationMeasurement)
}

// RecordRemediateDuration produces measurements for the RemediateDuration view.
func RecordRemediateDuration(ctx context.Context, status string, gvk schema.GroupVersionKind, startTime time.Time) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyStatus, status), tag.Upsert(KeyType, gvk.Kind))
	measurement := RemediateDuration.M(time.Since(startTime).Seconds())
	stats.Record(tagCtx, measurement)
}

// RecordResourceConflicts produces measurements for the ResourceConflicts view.
func RecordResourceConflicts(ctx context.Context, gvk schema.GroupVersionKind) {
	tagCtx, _ := tag.New(ctx, tag.Upsert(KeyType, gvk.Kind))
	measurement := ResourceConflicts.M(1)
	stats.Record(tagCtx, measurement)
}

// RecordInternalErrors produces measurements for the InternalErrors view.
func RecordInternalErrors(source string) {
	tagCtx, _ := tag.New(context.Background(), tag.Upsert(KeySource, source))
	measurement := InternalErrors.M(1)
	stats.Record(tagCtx, measurement)
}
