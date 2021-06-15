package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	csmetadata "github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
)

// IsInvalidAnnotation returns true if the annotation cannot be declared by users.
func IsInvalidAnnotation(k string) bool {
	return csmetadata.HasConfigSyncPrefix(k) && !csmetadata.IsSourceAnnotation(k)
}

// Annotations verifies that the given object does not have any invalid
// annotations.
func Annotations(obj ast.FileObject) status.Error {
	var invalid []string
	for k := range obj.GetAnnotations() {
		if IsInvalidAnnotation(k) {
			invalid = append(invalid, k)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalAnnotationDefinitionError(&obj, invalid)
	}
	return nil
}
