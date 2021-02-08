package hierarchical

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfigValidator returns a visitor that ensures all HierarchyConfig
// objects in system/ declare valid resources for hierarchical inheritance.
func HierarchyConfigValidator() parsed.ValidatorFunc {
	f := parsed.PerObjectVisitor(func(obj ast.FileObject) status.Error {
		if obj.GroupVersionKind() != kinds.HierarchyConfig() {
			return nil
		}
		s, err := obj.Structured()
		if err != nil {
			return err
		}
		for _, res := range s.(*v1.HierarchyConfig).Spec.Resources {
			// First validate HierarchyMode.
			switch res.HierarchyMode {
			case v1.HierarchyModeNone, v1.HierarchyModeInherit, v1.HierarchyModeDefault:
			default:
				return hierarchyconfig.IllegalHierarchyModeError(hc(groupKinds(res)[0], obj), res.HierarchyMode)
			}

			// Then validate resource GroupKinds.
			for _, gk := range groupKinds(res) {
				if !hierarchyconfig.AllowedInHierarchyConfigs(gk) {
					return hierarchyconfig.UnsupportedResourceInHierarchyConfigError(hc(gk, obj))
				}
			}
		}
		return nil
	})
	return parsed.ValidateSystemObjects(f)
}

func groupKinds(res v1.HierarchyConfigResource) []schema.GroupKind {
	kinds := res.Kinds
	if len(kinds) == 0 {
		kinds = []string{""}
	}
	gks := make([]schema.GroupKind, len(kinds))
	for i, kind := range kinds {
		gks[i] = schema.GroupKind{Group: res.Group, Kind: kind}
	}
	return gks
}

func hc(gk schema.GroupKind, res id.Resource) id.HierarchyConfig {
	return hierarchyconfig.FileGroupKindHierarchyConfig{
		GK:       gk,
		Resource: res,
	}
}
