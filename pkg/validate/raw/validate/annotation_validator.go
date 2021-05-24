package validate

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
)

// IsInvalidAnnotation returns true if the annotation cannot be declared by users.
func IsInvalidAnnotation(k string) bool {
	return hasConfigSyncPrefix(k) && !isSourceAnnotation(k)
}

// Annotations verifies that the given object does not have any invalid
// annotations.
func Annotations(obj ast.FileObject) status.Error {
	var invalid []string
	for a := range obj.GetAnnotations() {
		if IsInvalidAnnotation(a) {
			invalid = append(invalid, a)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalAnnotationDefinitionError(&obj, invalid)
	}
	return nil
}

// sourceAnnotations is a map of annotations that are valid to exist on objects
// in the source repository.
var sourceAnnotations = map[string]bool{
	v1.NamespaceSelectorAnnotationKey:         true,
	v1.LegacyClusterSelectorAnnotationKey:     true,
	v1alpha1.ClusterNameSelectorAnnotationKey: true,
	v1.ResourceManagementKey:                  true,
}

// isSourceAnnotation returns true if the annotation is a ConfigSync source
// annotation.
func isSourceAnnotation(s string) bool {
	return sourceAnnotations[s]
}

// hasConfigSyncPrefix returns true if the string begins with a ConfigSync
// annotation prefix.
func hasConfigSyncPrefix(s string) bool {
	return strings.HasPrefix(s, v1.ConfigManagementPrefix) || strings.HasPrefix(s, v1alpha1.ConfigSyncPrefix)
}
