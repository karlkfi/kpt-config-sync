package filesystem

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// extractHierarchyConfigs extracts HierarchyConfigs from FileObjects.
func extractHierarchyConfigs(objects []ast.FileObject) ([]*v1.HierarchyConfig, status.MultiError) {
	var configs []*v1.HierarchyConfig
	var errs status.MultiError
	for _, object := range objects {
		if object.GroupVersionKind() != kinds.HierarchyConfig() {
			continue
		}
		s, err := object.Structured()
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		configs = append(configs, s.(*v1.HierarchyConfig))
	}

	return configs, errs
}
