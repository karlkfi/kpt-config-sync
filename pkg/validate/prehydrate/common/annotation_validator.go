package common

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

// AnnotationValidator returns a visitor that verifies that no object has
// invalid annotations in the source repository.
func AnnotationValidator() parsed.ValidatorFunc {
	return parsed.ValidateAllObjects(parsed.PerObjectVisitor(validateAnnotations))
}

func validateAnnotations(obj ast.FileObject) status.Error {
	var invalid []string
	for a := range obj.GetAnnotations() {
		if hasConfigSyncPrefix(a) && !isSourceAnnotation(a) {
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
