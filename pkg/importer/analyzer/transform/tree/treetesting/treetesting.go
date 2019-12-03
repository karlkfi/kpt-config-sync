package treetesting

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
)

// BuildTree creates and populates an ast.Root with the provided objects.
// Assumes all objects are in the same top-level directory, and that top-level directory is the
// hierarchical config directory.
func BuildTree(t *testing.T, objects ...ast.FileObject) *ast.Root {
	t.Helper()

	return buildTree(t, &ast.Root{}, objects...)
}

// BuildFlatTree partitions FileObjects into a FlatRoot.
func BuildFlatTree(t *testing.T, objects ...ast.FileObject) *ast.FlatRoot {
	t.Helper()

	result := &ast.FlatRoot{}
	for _, object := range objects {
		switch object.Path.Split()[0] {
		case repo.SystemDir:
			result.SystemObjects = append(result.SystemObjects, object)
		case repo.ClusterRegistryDir:
			result.ClusterRegistryObjects = append(result.ClusterRegistryObjects, object)
		case repo.ClusterDir:
			result.ClusterObjects = append(result.ClusterObjects, object)
		case repo.NamespacesDir, repo.GCPResourceDir:
			result.NamespaceObjects = append(result.NamespaceObjects, object)
		default:
			t.Fatalf("test resource not in known top-level directory: %s", object.SlashPath())
		}
	}

	return result
}

func buildTree(t *testing.T, root *ast.Root, objects ...ast.FileObject) *ast.Root {
	flatRoot := BuildFlatTree(t, objects...)

	root.Accept(tree.NewSystemBuilderVisitor(flatRoot.SystemObjects))
	root.Accept(tree.NewClusterRegistryBuilderVisitor(flatRoot.ClusterRegistryObjects))
	root.Accept(tree.NewClusterBuilderVisitor(flatRoot.ClusterObjects))
	root.Accept(tree.NewBuilderVisitor(flatRoot.NamespaceObjects))

	return root
}
