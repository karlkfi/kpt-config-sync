package metrics

import (
	"os"
	"regexp"

	"go.opencensus.io/tag"
)

var (
	// KeyName groups metrics by the reconciler name. Possible values: root-reconciler, ns-reconciler-<namespace>
	KeyName, _ = tag.NewKey("name")

	// KeyNamespace groups metrics by the namespace
	KeyNamespace, _ = tag.NewKey("namespace")

	// KeyReconcilerType groups metrics by the reconciler type. Possible values: root, namespace.
	KeyReconcilerType, _ = tag.NewKey("reconciler")

	// KeyOperation groups metrics by their operation. Possible values: create, patch, update, delete.
	KeyOperation, _ = tag.NewKey("operation")

	// KeyComponent groups metrics by their component. Possible values: source, sync, rendering, readiness(from Resource Group Controller).
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

	// KeyContainer groups metrics by their container names. Possible values: reconciler, git-sync.
	KeyContainer, _ = tag.NewKey("container")

	// KeyResourceType groups metris by their resource types. Possible values: cpu, memory.
	KeyResourceType, _ = tag.NewKey("resource")
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
