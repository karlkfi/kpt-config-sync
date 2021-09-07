package metrics

import (
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/metrics"
	ocmetrics "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"go.opencensus.io/tag"
)

// ConfigSyncMetrics is a map from metric names to its measurements.
type ConfigSyncMetrics map[string][]Measurement

// Measurement is a recorded data point with a list of tags and a value.
type Measurement struct {
	Tags  []tag.Tag
	Value string
}

// GVKMetric is used for validating the count aggregated metrics that have a GVK
// type tag (`api_duration_seconds`, `apply_operations`, and `watches`).
type GVKMetric struct {
	GVK      string
	APIOp    string
	ApplyOps []Operation
	Watches  string
}

// Operation encapsulates an operation in the applier (create, update, delete)
// with its count value.
type Operation struct {
	Name  string
	Count int
}

// Validation evaluates a Measurement, returning an error if it fails validation.
type Validation func(metric Measurement) error

const (
	// MetricsPort is the port where metrics are exposed
	MetricsPort = ":8675"
	// OtelDeployment is name of the otel-collector deployment
	OtelDeployment = "deployment/otel-collector"
)

// ResourceCreated encapsulates the expected metric data when a new resource is created
// in Config Sync.
func ResourceCreated(gvk string) GVKMetric {
	return GVKMetric{
		GVK:   gvk,
		APIOp: "update",
		ApplyOps: []Operation{
			{Name: "update", Count: 1},
		},
		Watches: "1",
	}
}

// ResourcePatched encapsulates the expected metric data when an existing resource is
// patched in Config Sync.
func ResourcePatched(gvk string, count int) GVKMetric {
	return GVKMetric{
		GVK:      gvk,
		APIOp:    "update",
		ApplyOps: []Operation{{Name: "update", Count: count}},
		Watches:  "1",
	}
}

// ResourceDeleted encapsulates the expected metric data when a resource is deleted in
// Config Sync.
func ResourceDeleted(gvk string) GVKMetric {
	return GVKMetric{
		GVK:      gvk,
		APIOp:    "delete",
		ApplyOps: []Operation{{Name: "delete", Count: 1}},
		Watches:  "0",
	}
}

// ValidateReconcilerManagerMetrics validates the `reconcile_duration_seconds`
// metric from the reconciler manager.
func (csm ConfigSyncMetrics) ValidateReconcilerManagerMetrics() error {
	validation := hasTags([]tag.Tag{
		{Key: ocmetrics.KeyStatus, Value: "success"},
	})
	return csm.validateMetric(ocmetrics.ReconcileDurationView.Name, validation)
}

// ValidateReconcilerMetrics validates the non-error and non-GVK metrics produced
// by the reconcilers.
func (csm ConfigSyncMetrics) ValidateReconcilerMetrics(reconciler string, numResources int) error {
	// These metrics have non-deterministic values, so we just validate that the
	// metric exists for the correct reconciler and has a "success" status tag.
	metrics := []string{
		ocmetrics.ParseDurationView.Name,
		ocmetrics.ApplyDurationView.Name,
		ocmetrics.WatchManagerUpdatesDurationView.Name,
	}
	for _, m := range metrics {
		if err := csm.validateSuccessTag(reconciler, m); err != nil {
			return err
		}
	}
	return csm.validateDeclaredResources(reconciler, numResources)
}

// ValidateGVKMetrics validates all the metrics that have a GVK "type" tag key.
func (csm ConfigSyncMetrics) ValidateGVKMetrics(reconciler string, gvkMetric GVKMetric) error {
	if gvkMetric.APIOp != "" {
		if err := csm.validateAPICallDuration(reconciler, gvkMetric.APIOp, gvkMetric.GVK); err != nil {
			return err
		}
	}
	for _, applyOp := range gvkMetric.ApplyOps {
		if err := csm.validateApplyOperations(reconciler, applyOp.Name, gvkMetric.GVK, applyOp.Count); err != nil {
			return err
		}
	}
	if gvkMetric.Watches != "" {
		if err := csm.validateWatches(reconciler, gvkMetric.GVK, gvkMetric.Watches); err != nil {
			return err
		}
	}
	return csm.validateRemediateDuration(reconciler, gvkMetric.GVK)
}

// ValidateMetricsCommitApplied checks that the `last_apply_timestamp` metric has been
// recorded for a particular commit hash.
func (csm ConfigSyncMetrics) ValidateMetricsCommitApplied(commitHash string) error {
	validation := hasTags([]tag.Tag{
		{Key: ocmetrics.KeyCommit, Value: commitHash},
	})

	for _, metric := range csm[ocmetrics.LastApplyTimestampView.Name] {
		if validation(metric) == nil {
			return nil
		}
	}

	return errors.Errorf("Commit hash %s not found in config sync metrics", commitHash)
}

// ValidateErrorMetrics checks for the absence of all the error metrics except
// for the `reconciler_errors` metric. This metric is aggregated as a LastValue,
// so we check that the values are 0 instead.
func (csm ConfigSyncMetrics) ValidateErrorMetrics(reconciler string) error {
	metrics := []string{
		ocmetrics.ParseErrorsView.Name,
		ocmetrics.ResourceFightsView.Name,
		ocmetrics.ResourceConflictsView.Name,
		ocmetrics.InternalErrorsView.Name,
	}
	for _, m := range metrics {
		if _, ok := csm[m]; ok {
			return errors.Errorf("validating error metrics: expected no error metrics but found %v", m)
		}
	}
	return csm.ValidateReconcilerErrors(reconciler, 0, 0)
}

// ValidateParseError checks that the `parse_errors_total` metric is recorded with
// the correct error code tag and a value of at least 1.
func (csm ConfigSyncMetrics) ValidateParseError(reconciler, errorCode string) error {
	if _, ok := csm[ocmetrics.ParseErrorsView.Name]; ok {
		reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
		validations := []Validation{
			hasTags([]tag.Tag{
				{Key: reconcilerKey, Value: reconciler},
				{Key: ocmetrics.KeyErrorCode, Value: errorCode},
			}),
			valueGTE(1),
		}
		return csm.validateMetric(ocmetrics.ParseErrorsView.Name, validations...)
	}
	return nil
}

// ValidateReconcilerErrors checks that the `reconciler_errors` metric is recorded
// for the correct reconciler with the expected values for each of its component tags.
func (csm ConfigSyncMetrics) ValidateReconcilerErrors(reconciler string, sourceValue, syncValue int) error {
	if _, ok := csm[ocmetrics.ReconcilerErrorsView.Name]; ok {
		reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
		for _, measurement := range csm[ocmetrics.ReconcilerErrorsView.Name] {
			// If the measurement has a "source" tag, validate the values match.
			if hasTags([]tag.Tag{
				{Key: reconcilerKey, Value: reconciler},
				{Key: ocmetrics.KeyComponent, Value: "source"},
			})(measurement) == nil {
				if err := valueEquals(sourceValue)(measurement); err != nil {
					return err
				}
			}
			// If the measurement has a "sync" tag, validate the values match.
			if hasTags([]tag.Tag{
				{Key: reconcilerKey, Value: reconciler},
				{Key: ocmetrics.KeyComponent, Value: "sync"},
			})(measurement) == nil {
				if err := valueEquals(syncValue)(measurement); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ValidateReconcilerNonBlockingErrors checks that the `reconciler_non_blocking_errors` metric is recorded
// for the correct reconciler and error code, and checks the metric value is correct.
func (csm ConfigSyncMetrics) ValidateReconcilerNonBlockingErrors(reconciler, errorCode string, errorCount int) error {
	if _, ok := csm[ocmetrics.ReconcilerNonBlockingErrorsView.Name]; ok {
		validations := []Validation{
			hasTags([]tag.Tag{
				{Key: metrics.KeyReconciler, Value: reconciler},
				{Key: metrics.KeyErrorCode, Value: errorCode},
			}),
			valueEquals(errorCount),
		}
		return csm.validateMetric(ocmetrics.ReconcilerNonBlockingErrorsView.Name, validations...)
	}
	return nil
}

// validateSuccessTag checks that the metric is recorded for the correct reconciler
// and has a "success" tag value.
func (csm ConfigSyncMetrics) validateSuccessTag(reconciler, metric string) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	validation := hasTags([]tag.Tag{
		{Key: reconcilerKey, Value: reconciler},
		{Key: ocmetrics.KeyStatus, Value: "success"},
	})
	return csm.validateMetric(metric, validation)
}

// validateAPICallDuration checks that the `api_duration_seconds` metric is recorded
// and has the correct reconciler, operation, status, and type tags.
func (csm ConfigSyncMetrics) validateAPICallDuration(reconciler, operation, gvk string) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	validation := hasTags([]tag.Tag{
		{Key: reconcilerKey, Value: reconciler},
		{Key: ocmetrics.KeyOperation, Value: operation},
		{Key: ocmetrics.KeyStatus, Value: "success"},
		{Key: ocmetrics.KeyType, Value: gvk},
	})
	return errors.Wrapf(csm.validateMetric(ocmetrics.APICallDurationView.Name, validation), "%s %s operation", gvk, operation)
}

// validateDeclaredResources checks that the declared_resources metric is recorded
// and has the expected value.
func (csm ConfigSyncMetrics) validateDeclaredResources(reconciler string, value int) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	validations := []Validation{
		hasTags([]tag.Tag{{Key: reconcilerKey, Value: reconciler}}),
		valueEquals(value),
	}
	return csm.validateMetric(ocmetrics.DeclaredResourcesView.Name, validations...)
}

// validateApplyOperations checks that the `apply_operations` metric is recorded
// and has the correct reconciler, operation, status, and type tag values. Because
// controllers may fail and retry successfully, the recorded value of this metric may
// fluctuate, so we check that it is greater than or equal to the expected value.
func (csm ConfigSyncMetrics) validateApplyOperations(reconciler, operation, gvk string, value int) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	validations := []Validation{
		hasTags([]tag.Tag{
			{Key: reconcilerKey, Value: reconciler},
			{Key: ocmetrics.KeyOperation, Value: operation},
			{Key: ocmetrics.KeyStatus, Value: "success"},
			{Key: ocmetrics.KeyType, Value: gvk},
		}),
		valueGTE(value),
	}
	return errors.Wrapf(csm.validateMetric(ocmetrics.ApplyOperationsView.Name, validations...), "%s %s operation", gvk, operation)
}

// validateWatches checks that the `watches` metric is recorded with
// the correct gvk type tag and has the expected value.
func (csm ConfigSyncMetrics) validateWatches(reconciler, gvk, value string) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	mv, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	validations := []Validation{
		hasTags([]tag.Tag{
			{Key: reconcilerKey, Value: reconciler},
			{Key: ocmetrics.KeyType, Value: gvk},
		}),
		valueEquals(mv),
	}
	return errors.Wrap(csm.validateMetric(ocmetrics.WatchesView.Name, validations...), gvk)
}

// validateRemediateDuration checks that the `remediate_duration_seconds` metric
// is recorded and has the correct status and type tags.
func (csm ConfigSyncMetrics) validateRemediateDuration(reconciler, gvk string) error {
	reconcilerKey, _ := tag.NewKey(strings.ReplaceAll(reconciler, "-", "_"))
	validations := []Validation{
		hasTags([]tag.Tag{
			{Key: reconcilerKey, Value: reconciler},
			{Key: ocmetrics.KeyStatus, Value: "success"},
			{Key: ocmetrics.KeyType, Value: gvk},
		}),
	}
	return errors.Wrap(csm.validateMetric(ocmetrics.RemediateDurationView.Name, validations...), gvk)
}

// validateMetric checks that at least one measurement from the metric passes all the validations.
func (csm ConfigSyncMetrics) validateMetric(name string, validations ...Validation) error {
	var errs status.MultiError
	allValidated := func(entry Measurement, vs []Validation) bool {
		for _, v := range vs {
			err := v(entry)
			if err != nil {
				errs = status.Append(errs, err)
				return false
			}
		}
		return true
	}

	if entries, ok := csm[name]; ok {
		for _, e := range entries {
			if allValidated(e, validations) {
				return nil
			}
		}
		return errors.Errorf("metric validations failed for metric %s: %v", name, errs)
	}
	return errors.Errorf("validating metric: metric %s not recorded", name)
}

// hasTags checks that the measurement contains all the expected tags.
func hasTags(tags []tag.Tag) Validation {
	return func(metric Measurement) error {
		contains := func(tts []tag.Tag, t tag.Tag) bool {
			for _, tt := range tts {
				if tt == t {
					return true
				}
			}
			return false
		}

		for _, t := range tags {
			if !contains(metric.Tags, t) {
				return errors.Errorf("expected tag %v but not found in measurement", t)
			}
		}
		return nil
	}
}

// valueEquals checks that the measurement is recorded with the expected value.
func valueEquals(value int) Validation {
	return func(metric Measurement) error {
		mv, err := strconv.Atoi(metric.Value)
		if err != nil {
			return err
		}
		if !cmp.Equal(mv, value) {
			return errors.Errorf("unexpected metric value (tags: %v), got %v but expected %v", metric.Tags, metric.Value, value)
		}
		return nil
	}
}

// valueGTE checks that the measurement value is greater than or equal to the expected value.
func valueGTE(value int) Validation {
	return func(metric Measurement) error {
		mv, err := strconv.Atoi(metric.Value)
		if err != nil {
			return err
		}
		if mv < value {
			return errors.Errorf("unexpected metric value (tags: %v), got %v but expected "+
				"a value greater than or equal to %v", metric.Tags, mv, value)
		}
		return nil
	}
}
