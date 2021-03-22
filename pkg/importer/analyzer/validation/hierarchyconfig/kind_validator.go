package hierarchyconfig

import (
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UnsupportedResourceInHierarchyConfigErrorCode is the error code for UnsupportedResourceInHierarchyConfigError
const UnsupportedResourceInHierarchyConfigErrorCode = "1041"

var unsupportedResourceInHierarchyConfigError = status.NewErrorBuilder(UnsupportedResourceInHierarchyConfigErrorCode)

// UnsupportedResourceInHierarchyConfigError reports that config management is unsupported for a Resource defined in a HierarchyConfig.
func UnsupportedResourceInHierarchyConfigError(config client.Object, gk schema.GroupKind) status.Error {
	return unsupportedResourceInHierarchyConfigError.
		Sprintf("The %q APIResource MUST NOT be declared in a HierarchyConfig:",
			gk.String()).
		BuildWithResources(config)
}
