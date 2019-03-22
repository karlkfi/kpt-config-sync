package treetesting

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/util/discovery"
)

// BuildTree creates and populates an ast.Root with the provided objects.
// Assumes all objects are in the same top-level directory, and that top-level directory is the
// hierarchical policy directory.
func BuildTree(t *testing.T, objects ...ast.FileObject) *ast.Root {
	t.Helper()

	return buildTree(t, &ast.Root{}, objects...)
}

// BuildTreeWithAPIInfo builds the tree and sets the APIInfo in the root node.
func BuildTreeWithAPIInfo(t *testing.T, apiInfo *discovery.APIInfo, objects ...ast.FileObject) *ast.Root {
	t.Helper()

	root := &ast.Root{}
	discovery.AddAPIInfo(root, apiInfo)

	return buildTree(t, root, objects...)
}

func buildTree(t *testing.T, root *ast.Root, objects ...ast.FileObject) *ast.Root {
	// TODO: Move this to transforming visitors.
	var namespaceObjects []ast.FileObject
	var sytemObjects []ast.FileObject
	var clusterObjects []ast.FileObject
	for _, object := range objects {
		switch object.Path.Split()[0] {
		case repo.SystemDir:
			sytemObjects = append(sytemObjects, object)
		case repo.ClusterRegistryDir:
			root.ClusterRegistryObjects = append(root.ClusterRegistryObjects, &ast.ClusterRegistryObject{FileObject: object})
		case repo.ClusterDir:
			clusterObjects = append(clusterObjects, object)
		case repo.NamespacesDir:
			namespaceObjects = append(namespaceObjects, object)
		default:
			t.Fatalf("test resource not in known top-level directory: %s", object.SlashPath())
		}
	}

	root.Accept(tree.NewSystemBuilderVisitor(sytemObjects))
	root.Accept(tree.NewClusterBuilderVisitor(clusterObjects))
	root.Accept(tree.NewBuilderVisitor(namespaceObjects))

	return root
}
