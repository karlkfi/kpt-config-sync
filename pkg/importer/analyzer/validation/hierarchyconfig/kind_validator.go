package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewHierarchyConfigKindValidator returns a Visitor that ensures only supported Resource Kinds are declared in
// HierarchyConfigs.
func NewHierarchyConfigKindValidator(enableCRDs bool) ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch h := o.Object.(type) {
		case *v1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o).flatten() {
				if err := ValidateKinds(gkc, enableCRDs); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateKinds ensures that only supported Resource Kinds are declared in HierarchyConfigs.
func ValidateKinds(config FileGroupKindHierarchyConfig, enableCRDs bool) status.MultiError {
	if AllowedInHierarchyConfigs(config.GroupKind(), enableCRDs) {
		return nil
	}
	return status.From(vet.UnsupportedResourceInHierarchyConfigError{
		HierarchyConfig: config,
	})
}

// AllowedInHierarchyConfigs returns true if the passed GroupKind is allowed to be declared in HierarchyConfigs.
func AllowedInHierarchyConfigs(gk schema.GroupKind, enableCRDs bool) bool {
	return !unsupportedHierarchyConfigResources(enableCRDs)[gk] && gk.Group != configmanagement.GroupName && gk.Kind != ""
}

// unsupportedHierarchyConfigResources returns a map of each type where syncing is explicitly not supported.
func unsupportedHierarchyConfigResources(enableCRDs bool) map[schema.GroupKind]bool {
	m := map[schema.GroupKind]bool{
		kinds.Namespace().GroupKind():        true,
		kinds.ConfigManagement().GroupKind(): true,
	}
	if !enableCRDs {
		m[kinds.CustomResourceDefinition().GroupKind()] = true
	}
	return m
}
