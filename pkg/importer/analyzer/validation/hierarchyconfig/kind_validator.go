package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewHierarchyConfigKindValidator returns a Visitor that ensures only supported Resource Kinds are declared in
// HierarchyConfigs.
func NewHierarchyConfigKindValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) *status.MultiError {
		switch h := o.Object.(type) {
		case *v1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o).flatten() {
				if err := ValidateKinds(gkc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateKinds ensures that only supported Resource Kinds are declared in HierarchyConfigs.
func ValidateKinds(config FileGroupKindHierarchyConfig) *status.MultiError {
	if AllowedInHierarchyConfigs(config.GroupKind()) {
		return nil
	}
	return status.From(vet.UnsupportedResourceInHierarchyConfigError{
		HierarchyConfig: config,
	})
}

// AllowedInHierarchyConfigs returns true if the passed GroupKind is allowed to be declared in HierarchyConfigs.
func AllowedInHierarchyConfigs(gk schema.GroupKind) bool {
	return !unsupportedHierarchyConfigResources()[gk] && gk.Group != configmanagement.GroupName && gk.Kind != ""
}

// unsupportedHierarchyConfigResources returns a map of each type where syncing is explicitly not supported.
func unsupportedHierarchyConfigResources() map[schema.GroupKind]bool {
	return map[schema.GroupKind]bool{
		kinds.CustomResourceDefinition().GroupKind(): true,
		kinds.Namespace().GroupKind():                true,
	}
}
