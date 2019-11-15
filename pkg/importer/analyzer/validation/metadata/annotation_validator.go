package metadata

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// inputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var inputAnnotations = map[string]bool{
	v1.NamespaceSelectorAnnotationKey: true,
	v1.ClusterSelectorAnnotationKey:   true,
	v1.ResourceManagementKey:          true,
}

// isInputAnnotation returns true if the annotation is a Nomos input annotation.
func isInputAnnotation(s string) bool {
	return inputAnnotations[s]
}

// hasConfigManagementPrefix returns true if the string begins with the Nomos annotation prefix.
func hasConfigManagementPrefix(s string) bool {
	return strings.HasPrefix(s, v1.ConfigManagementPrefix)
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

// NewManagedAnnotationValidator validates the value of the management annotation label.
func NewManagedAnnotationValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			return ValidManagementAnnotation(o)
		})
}

// ValidManagementAnnotation returns an Error if the user-specified Managment annotation is invalid.
func ValidManagementAnnotation(o ast.FileObject) status.Error {
	value, found := o.GetAnnotations()[v1.ResourceManagementKey]
	if found && (value != v1.ResourceManagementDisabled) {
		return IllegalManagementAnnotationError(&o, value)
	}
	return nil
}

// IllegalManagementAnnotationErrorCode is the error code for IllegalManagementAnnotationError.
const IllegalManagementAnnotationErrorCode = "1005"

var illegalManagementAnnotationError = status.NewErrorBuilder(IllegalManagementAnnotationErrorCode)

// IllegalManagementAnnotationError represents an illegal management annotation value.
// Error implements error.
func IllegalManagementAnnotationError(resource id.Resource, value string) status.Error {
	return illegalManagementAnnotationError.
		Sprintf("Config has invalid management annotation %s=%s. Must be %s or unset.",
			v1.ResourceManagementKey, value, v1.ResourceManagementDisabled).
		BuildWithResources(resource)
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
		Sprintf("Configs MUST NOT declare unsupported annotations starting with %q. "+
			"The config has invalid annotations: %s",
			v1.ConfigManagementPrefix, a).
		BuildWithResources(resource)
}
