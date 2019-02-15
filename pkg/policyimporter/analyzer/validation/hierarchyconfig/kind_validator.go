package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewHierarchyConfigKindValidator returns a Visitor that ensures only supported Resource Kinds are declared in
// HierarchyConfigs.
func NewHierarchyConfigKindValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) error {
		switch h := o.Object.(type) {
		case *v1alpha1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o.Relative).flatten() {
				if err := ValidateKinds(gkc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateKinds ensures that only supported Resource Kinds are declared in HierarchyConfigs.
func ValidateKinds(config FileGroupKindHierarchyConfig) error {
	if AllowedInHierarchyConfigs(config.GroupKind()) {
		return nil
	}
	return vet.UnsupportedResourceInHierarchyConfigError{
		HierarchyConfig: config,
	}
}

// AllowedInHierarchyConfigs returns true if the passed GroupKind is allowed to be declared in HierarchyConfigs.
func AllowedInHierarchyConfigs(gk schema.GroupKind) bool {
	return !unsupportedHierarchyConfigResources()[gk] && gk.Group != policyhierarchy.GroupName && gk.Kind != ""
}

// unsupportedHierarchyConfigResources returns a map of each type where syncing is explicitly not supported.
func unsupportedHierarchyConfigResources() map[schema.GroupKind]bool {
	return map[schema.GroupKind]bool{
		kinds.CustomResourceDefinition().GroupKind(): true,
		kinds.Namespace().GroupKind():                true,
	}
}
