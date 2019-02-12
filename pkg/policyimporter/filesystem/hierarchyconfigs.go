package filesystem

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
)

// extractHierarchyConfigs extracts HierarchyConfigs from FileObjects.
func extractHierarchyConfigs(objects []ast.FileObject) []*v1alpha1.HierarchyConfig {
	var configs []*v1alpha1.HierarchyConfig
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.HierarchyConfig:
			configs = append(configs, o)
		}
	}

	return configs
}
