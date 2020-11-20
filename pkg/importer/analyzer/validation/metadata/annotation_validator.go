package metadata

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// inputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var inputAnnotations = map[string]bool{
	v1.NamespaceSelectorAnnotationKey:         true,
	v1.LegacyClusterSelectorAnnotationKey:     true,
	v1alpha1.ClusterNameSelectorAnnotationKey: true,
	v1.ResourceManagementKey:                  true,
}

// isInputAnnotation returns true if the annotation is a Nomos input annotation.
func isInputAnnotation(s string) bool {
	return inputAnnotations[s]
}

// hasConfigManagementPrefix returns true if the string begins with the Nomos annotation prefix.
func hasConfigManagementPrefix(s string) bool {
	return strings.HasPrefix(s, v1.ConfigManagementPrefix) || strings.HasPrefix(s, v1alpha1.ConfigSyncPrefix)
}

// NewAnnotationValidator validates the annotations of every object.
func NewAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			var errors []string
			for a := range o.GetAnnotations() {
				if !isInputAnnotation(a) && hasConfigManagementPrefix(a) {
					errors = append(errors, a)
				}
			}
			if errors != nil {
				return IllegalAnnotationDefinitionError(&o, errors)
			}
			return nil
		})
}

// IllegalAnnotationDefinitionErrorCode is the error code for IllegalAnnotationDefinitionError
const IllegalAnnotationDefinitionErrorCode = "1010"

var illegalAnnotationDefinitionError = status.NewErrorBuilder(IllegalAnnotationDefinitionErrorCode)

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
func IllegalAnnotationDefinitionError(resource id.Resource, annotations []string) status.Error {
	sort.Strings(annotations) // ensure deterministic annotation order
	annotations2 := make([]string, len(annotations))
	for i, annotation := range annotations {
		annotations2[i] = fmt.Sprintf("%q", annotation)
	}
	a := strings.Join(annotations2, ", ")
	return illegalAnnotationDefinitionError.
		Sprintf("Configs MUST NOT declare unsupported annotations starting with %q or %q. "+
			"The config has invalid annotations: %s",
			v1.ConfigManagementPrefix, v1alpha1.ConfigSyncPrefix, a).
		BuildWithResources(resource)
}
