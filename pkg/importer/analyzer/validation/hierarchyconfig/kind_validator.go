package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewHierarchyConfigKindValidator returns a Visitor that ensures only supported Resource Kinds are declared in
// HierarchyConfigs.
func NewHierarchyConfigKindValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		if o.GroupVersionKind() != kinds.HierarchyConfig() {
			return nil
		}
		s, err := o.Structured()
		if err != nil {
			return err
		}
		hc := s.(*v1.HierarchyConfig)
		for _, gkc := range newFileHierarchyConfig(hc, o).flatten() {
			if err := validateKinds(gkc); err != nil {
				return err
			}
		}
		return nil
	})
}

// ValidateKinds ensures that only supported Resource Kinds are declared in HierarchyConfigs.
func validateKinds(config FileGroupKindHierarchyConfig) status.MultiError {
	if AllowedInHierarchyConfigs(config.GroupKind()) {
		return nil
	}
	return UnsupportedResourceInHierarchyConfigError(config)
}

// AllowedInHierarchyConfigs returns true if the passed GroupKind is allowed to be declared in HierarchyConfigs.
func AllowedInHierarchyConfigs(gk schema.GroupKind) bool {
	return !unsupportedHierarchyConfigResources()[gk] && gk.Group != configmanagement.GroupName && gk.Kind != ""
}

// unsupportedHierarchyConfigResources returns a map of each type where syncing is explicitly not supported.
func unsupportedHierarchyConfigResources() map[schema.GroupKind]bool {
	m := map[schema.GroupKind]bool{
		kinds.Namespace().GroupKind():        true,
		kinds.ConfigManagement().GroupKind(): true,
	}

	return m
}

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
