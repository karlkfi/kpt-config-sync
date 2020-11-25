package metrics

import "go.opencensus.io/tag"

var (
	// KeyScope groups metrics by their scope. Possible values: root-sync, repo-sync.
	KeyScope, _ = tag.NewKey("scope")

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

	// KeySource groups metrics by their source. Possible values: parser, differ, remediator.
	KeySource, _ = tag.NewKey("source")
)

// StatusTagKey returns a string representation of the error, if it exists, otherwise success.
func StatusTagKey(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}
