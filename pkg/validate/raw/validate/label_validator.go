package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
)

// IsInvalidLabel returns true if the label cannot be declared by users.
func IsInvalidLabel(k string) bool {
	return HasConfigSyncPrefix(k)
}

// Labels verifies that the given object does not have any invalid labels.
func Labels(obj ast.FileObject) status.Error {
	var invalid []string
	for l := range obj.GetLabels() {
		if IsInvalidLabel(l) {
			invalid = append(invalid, l)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalLabelDefinitionError(&obj, invalid)
	}
	return nil
}
