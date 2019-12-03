package selectors

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
)

// AnnotateClusterName sets the value of the cluster-name annotation to clusterName.
func AnnotateClusterName(clusterName string, objects []ast.FileObject) []ast.FileObject {
	if clusterName == "" {
		return objects
	}

	for _, object := range objects {
		core.Annotation(v1.ClusterNameAnnotationKey, clusterName)(object)
	}
	return objects
}
