package metrics

import (
	"os"
	"regexp"
	"strings"

	"go.opencensus.io/tag"
)

var (
	// KeyReconciler is a dynamic key where both the key and value are set to
	// the name of the reconciler the metric is emitted from.
	// Possible values: root_reconciler, ns_reconciler_<namespace>
	//
	// We need to use a dynamic label so that the OpenTelemetry Collector can
	// differentiate between the metrics emitted by the different reconcilers.
	// Otherwise, metrics will be randomly sampled:
	// https://github.com/open-telemetry/opentelemetry-collector/issues/1076.
	KeyReconciler, _ = tag.NewKey(strings.ReplaceAll(ReconcilerTagKey(), "-", "_"))

	// KeyOperation groups metrics by their operation. Possible values: create, patch, update, delete.
	KeyOperation, _ = tag.NewKey("operation")

	// KeyComponent groups metrics by their component. Possible values: source, sync.
	KeyComponent, _ = tag.NewKey("component")

	// KeyErrorCode groups metrics by their error code.
	KeyErrorCode, _ = tag.NewKey("errorcode")

	// KeyStatus groups metrics by their status. Possible values: success, error.
	KeyStatus, _ = tag.NewKey("status")

	// KeyType groups metrics by their resource GVK.
	KeyType, _ = tag.NewKey("type")

	// KeyInternalErrorSource groups the InternalError metrics by their source. Possible values: parser, differ, remediator.
	KeyInternalErrorSource, _ = tag.NewKey("source")

	// KeyParserSource groups the metrics for the parser by their source. Possible values: read, parse, update.
	KeyParserSource, _ = tag.NewKey("source")

	// KeyTrigger groups metrics by their trigger. Possible values: retry, watchUpdate, managementConflict, resync, reimport.
	KeyTrigger, _ = tag.NewKey("trigger")

	// KeyCommit groups metrics by their git commit. Even though this tag has a high cardinality,
	// it is only used by the `last_sync_timestamp` and `last_apply_timestamp` metrics.
	// These are both aggregated as LastValue metrics so the number of recorded values will always be
	// at most 1 per git commit.
	KeyCommit, _ = tag.NewKey("commit")
)

// StatusTagKey returns a string representation of the error, if it exists, otherwise success.
func StatusTagKey(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}

// ReconcilerTagKey filters the reconciler name from the pod name that is exposed via the Downward API
// (https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#the-downward-api).
// If the regex filter fails, the entire pod name is returned.
func ReconcilerTagKey() string {
	podName := os.Getenv("RECONCILER_NAME")
	regex := regexp.MustCompile(`(?:([a-z0-9]+(?:-[a-z0-9]+)*))-[a-z0-9]+-(?:[a-z0-9]+)`)
	ss := regex.FindStringSubmatch(podName)
	if ss != nil {
		return ss[1]
	}
	return podName
}
