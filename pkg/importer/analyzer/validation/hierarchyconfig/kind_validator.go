package hierarchyconfig

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedResourceInHierarchyConfigErrorCode is the error code for UnsupportedResourceInHierarchyConfigError
const UnsupportedResourceInHierarchyConfigErrorCode = "1041"

var unsupportedResourceInHierarchyConfigError = status.NewErrorBuilder(UnsupportedResourceInHierarchyConfigErrorCode)

// UnsupportedResourceInHierarchyConfigError reports that config management is unsupported for a Resource defined in a HierarchyConfig.
func UnsupportedResourceInHierarchyConfigError(config id.HierarchyConfig) status.Error {
	gk := config.GroupKind()
	return unsupportedResourceInHierarchyConfigError.
		Sprintf("The %q APIResource MUST NOT be declared in a HierarchyConfig:",
			gk.String()).
		BuildWithResources(config)
}
