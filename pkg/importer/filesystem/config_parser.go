package filesystem

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// GetSyncedCRDs is a callback that can be used to list the CRDs on a cluster.
// Only called if the parsing logic actually requires it, i.e. if a repository
// declares a non-base Kubernetes type, doesn't have a CRD for it, and the
// caller has not disabled API Server checks.
type GetSyncedCRDs func() ([]*v1beta1.CustomResourceDefinition, status.MultiError)

// NoSyncedCRDs is a no-op GetSyncedCRDs.
// CSMR doesn't use ClusterConfigs, so it is unnecessary.
var NoSyncedCRDs GetSyncedCRDs = func() ([]*apiextensionsv1beta1.CustomResourceDefinition, status.MultiError) {
	return nil, nil
}

// ConfigParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type ConfigParser interface {
	Parse(clusterName string,
		enableAPIServerChecks bool,
		getSyncedCRDs GetSyncedCRDs,
		filePaths FilePaths,
	) ([]core.Object, status.MultiError)

	// ReadClusterRegistryResources returns the list of Clusters contained in the repo.
	ReadClusterRegistryResources(filePaths FilePaths) []ast.FileObject
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
