package metrics

import "go.opencensus.io/tag"

var (
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

	// KeyParserSource groups the metrics for the parser by their source. Possible values: read, parse.
	KeyParserSource, _ = tag.NewKey("source")

	// KeyTrigger groups metrics by their trigger. Possible values: retry, watchUpdate, managementConflict, resync, reimport.
	KeyTrigger, _ = tag.NewKey("trigger")
)

// StatusTagKey returns a string representation of the error, if it exists, otherwise success.
func StatusTagKey(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}
