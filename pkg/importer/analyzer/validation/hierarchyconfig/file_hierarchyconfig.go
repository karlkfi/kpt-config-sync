package hierarchyconfig

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FileGroupKindHierarchyConfig Identifies a Group/Kind definition in a HierarchyConfig.
type FileGroupKindHierarchyConfig struct {
	// GK is the Group/Kind which the HierarchyConfig defined.
	GK schema.GroupKind
	// HierarchyMode is the hierarchy mode which the HierarchyConfig defined for the Kind.
	HierarchyMode v1.HierarchyModeType
	// Resource is the source file defining the HierarchyConfig.
	id.Resource
}

var _ id.HierarchyConfig = FileGroupKindHierarchyConfig{}

// GroupKind implements vet.HierarchyConfig
func (s FileGroupKindHierarchyConfig) GroupKind() schema.GroupKind {
	return s.GK
}
