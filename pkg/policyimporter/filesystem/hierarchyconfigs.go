package filesystem

import (
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// extractHierarchyConfigs extracts HierarchyConfigs from FileObjects.
func extractHierarchyConfigs(objects []ast.FileObject) []*v1.HierarchyConfig {
	var configs []*v1.HierarchyConfig
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1.HierarchyConfig:
			configs = append(configs, o)
		}
	}

	return configs
}
