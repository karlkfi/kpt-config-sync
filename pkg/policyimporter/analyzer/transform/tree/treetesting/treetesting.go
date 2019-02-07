package treetesting

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
)

// BuildTree creates and populates an ast.Root with the provided objects.
// Assumes all objects are in the same top-level directory, and that top-level directory is the
// hierarchical policy directory.
func BuildTree(t *testing.T, objects ...ast.FileObject) *ast.Root {
	t.Helper()

	root := &ast.Root{}

	// TODO: Move this to transforming visitors.
	var namespaceObjects []ast.FileObject
	var sytemObjects []ast.FileObject
	var clusterObjects []ast.FileObject
	for _, object := range objects {
		switch object.Relative.Split()[0] {
		case repo.SystemDir:
			sytemObjects = append(sytemObjects, object)
		case repo.ClusterRegistryDir:
			if root.ClusterRegistry == nil {
				root.ClusterRegistry = &ast.ClusterRegistry{}
			}
			root.ClusterRegistry.Objects = append(root.ClusterRegistry.Objects, &ast.ClusterRegistryObject{FileObject: object})
		case repo.ClusterDir:
			clusterObjects = append(clusterObjects, object)
		case repo.NamespacesDir:
			namespaceObjects = append(namespaceObjects, object)
		default:
			t.Fatalf("test resource not in known top-level directory: %s", object.RelativeSlashPath())
		}
	}

	root.Accept(tree.NewSystemBuilderVisitor(sytemObjects))
	root.Accept(tree.NewClusterBuilderVisitor(clusterObjects))
	root.Accept(tree.NewBuilderVisitor(namespaceObjects))

	return root
}
