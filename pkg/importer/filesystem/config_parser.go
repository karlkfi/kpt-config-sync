package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type ConfigParser interface {
	Parse(filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError)

	// ReadClusterRegistryResources returns the list of Clusters contained in the repo.
	ReadClusterRegistryResources(filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError)
}

// AsCoreObjects converts a slice of FileObjects to a slice of client.Objects.
func AsCoreObjects(fos []ast.FileObject) []client.Object {
	result := make([]client.Object, len(fos))
	for i, fo := range fos {
		result[i] = fo.Unstructured
	}
	return result
}
