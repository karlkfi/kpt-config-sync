package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
)

// Labels verifies that the given object does not have any invalid labels.
func Labels(obj ast.FileObject) status.Error {
	var invalid []string
	for l := range obj.GetLabels() {
		if hasConfigSyncPrefix(l) {
			invalid = append(invalid, l)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalLabelDefinitionError(&obj, invalid)
	}
	return nil
}
