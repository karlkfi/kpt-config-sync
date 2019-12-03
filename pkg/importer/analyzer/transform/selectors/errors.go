package selectors

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectHasUnknownSelectorCode is the error code for ObjectHasUnknownClusterSelector
const ObjectHasUnknownSelectorCode = "1013"

var objectHasUnknownSelector = status.NewErrorBuilder(ObjectHasUnknownSelectorCode)

// InvalidSelectorErrorCode is the error code for InvalidSelectorError
const InvalidSelectorErrorCode = "1014" // TODO: Must refactor to use properly

var invalidSelectorError = status.NewErrorBuilder(InvalidSelectorErrorCode)

// InvalidSelectorError reports that a ClusterSelector or NamespaceSelector is
// invalid.
// To be renamed in refactoring that removes above error.
func InvalidSelectorError(selector id.Resource, cause error) status.Error {
	return invalidSelectorError.Sprintf("%s has validation errors that must be corrected", selector.GroupVersionKind().Kind).Wrap(cause).BuildWithResources(selector)
}

// EmptySelectorError reports that a ClusterSelector or NamespaceSelector is
// invalid because it is empty.
func EmptySelectorError(selector id.Resource) status.Error {
	return invalidSelectorError.Sprintf("%ss MUST define spec.selector", selector.GroupVersionKind().Kind).BuildWithResources(selector)
}
