package hierarchyconfig

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/policyimporter/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FileHierarchyConfig extends v1alpha1.HierarchyConfig to include the path to the file in the repo.
type FileHierarchyConfig struct {
	*v1.HierarchyConfig
	// Resource is the source file defining the HierarchyConfig.
	id.Resource
}

// NewFileHierarchyConfig creates a new FileHierarchyConfig from a HierarchyConfig Resource and the source file declaring
// the HierarchyConfig.
func NewFileHierarchyConfig(config *v1.HierarchyConfig, resource id.Resource) FileHierarchyConfig {
	return FileHierarchyConfig{HierarchyConfig: config, Resource: resource}
}

// flatten returns a list of all GroupKinds defined in the HierarchyConfig and their hierarchy modes.
func (c FileHierarchyConfig) flatten() []FileGroupKindHierarchyConfig {
	var result []FileGroupKindHierarchyConfig
	for _, resource := range c.Spec.Resources {
		if len(resource.Kinds) == 0 {
			result = append(result, FileGroupKindHierarchyConfig{
				groupKind:     schema.GroupKind{Group: resource.Group},
				HierarchyMode: resource.HierarchyMode,
				Resource:      c.Resource,
			})
		} else {
			for _, kind := range resource.Kinds {
				result = append(result, FileGroupKindHierarchyConfig{
					groupKind:     schema.GroupKind{Group: resource.Group, Kind: kind},
					HierarchyMode: resource.HierarchyMode,
					Resource:      c.Resource,
				})
			}
		}
	}
	return result
}

// FileGroupKindHierarchyConfig Identifies a Group/Kind definition in a HierarchyConfig.
type FileGroupKindHierarchyConfig struct {
	// groupKind is the Group/Kind which the HierarchyConfig defined.
	groupKind schema.GroupKind
	// HierarchyMode is the hierarchy mode which the HierarchyConfig defined for the Kind.
	HierarchyMode v1.HierarchyModeType
	// Resource is the source file defining the HierarchyConfig.
	id.Resource
}

var _ id.HierarchyConfig = FileGroupKindHierarchyConfig{}

// GroupKind implements vet.HierarchyConfig
func (s FileGroupKindHierarchyConfig) GroupKind() schema.GroupKind {
	return s.groupKind
}
