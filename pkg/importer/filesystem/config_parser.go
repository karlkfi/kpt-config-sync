package filesystem

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// ConfigParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type ConfigParser interface {
	Parse(clusterName string, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper discovery.BuildScoperFunc, filePaths reader.FilePaths) ([]core.Object, status.MultiError)

	// ReadClusterRegistryResources returns the list of Clusters contained in the repo.
	ReadClusterRegistryResources(filePaths reader.FilePaths) []ast.FileObject
}

// AsCoreObjects converts a slice of FileObjects to a slice of core.Objects.
func AsCoreObjects(fos []ast.FileObject) []core.Object {
	result := make([]core.Object, len(fos))
	for i, fo := range fos {
		result[i] = fo.Object
	}
	return result
}

// AsFileObjects converts a slice of core.Objects to a slice of FileObjects.
func AsFileObjects(os []core.Object) []ast.FileObject {
	result := make([]ast.FileObject, len(os))
	for i, o := range os {
		result[i] = *ast.ParseFileObject(o)
	}
	return result
}
