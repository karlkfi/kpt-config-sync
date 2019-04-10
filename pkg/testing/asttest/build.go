package asttest

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/status"
)

// Build creates and populates an ast.Root with the provided objects.
// Assumes all objects are in the same top-level directory, and that top-level directory is the
// hierarchical policy directory.
func Build(t *testing.T, opts ...ast.BuildOpt) *ast.Root {
	t.Helper()

	root, err := ast.Build(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// Objects adds a list of objects to the AST using the default builders. Each object must be in a
// recognized top-level directory.
func Objects(objects ...ast.FileObject) ast.BuildOpt {
	var systemObjects []ast.FileObject
	return func(root *ast.Root) status.MultiError {
		var clusterObjects []ast.FileObject
		var clusterRegistryObjects []ast.FileObject
		var namespaceObjects []ast.FileObject
		for _, object := range objects {
			switch object.Path.Split()[0] {
			case repo.SystemDir:
				systemObjects = append(systemObjects, object)
			case repo.ClusterRegistryDir:
				clusterRegistryObjects = append(clusterRegistryObjects, object)
			case repo.ClusterDir:
				clusterObjects = append(clusterObjects, object)
			case repo.NamespacesDir:
				namespaceObjects = append(namespaceObjects, object)
			default:
				return status.From(vet.InternalErrorf("test resource not in known top-level directory: %s", object.SlashPath()))
			}
		}

		systemBuilder := tree.NewSystemBuilderVisitor(systemObjects)
		root.Accept(systemBuilder)
		if err := systemBuilder.Error(); err != nil {
			return err
		}

		clusterBuilder := tree.NewClusterBuilderVisitor(clusterObjects)
		root.Accept(clusterBuilder)
		if err := clusterBuilder.Error(); err != nil {
			return err
		}

		clusterRegistryBuilder := tree.NewClusterRegistryBuilderVisitor(clusterRegistryObjects)
		root.Accept(clusterRegistryBuilder)
		if err := clusterRegistryBuilder.Error(); err != nil {
			return err
		}

		namespaceVisitor := tree.NewBuilderVisitor(namespaceObjects)
		root.Accept(namespaceVisitor)
		return namespaceVisitor.Error()
	}
}
