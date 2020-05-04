package filesystem

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// GetSyncedCRDs is a callback that can be used to list the CRDs on a cluster.
// Only called if the parsing logic actually requires it, i.e. if a repository
// declares a non-base Kubernetes type, doesn't have a CRD for it, and the
// caller has not disabled API Server checks.
type GetSyncedCRDs func() ([]*v1beta1.CustomResourceDefinition, status.MultiError)

// ConfigParser defines the minimum interface required for Reconciler to use a Parser to read
// configs from a filesystem.
type ConfigParser interface {
	Parse(clusterName string,
		enableAPIServerChecks bool,
		getSyncedCRDs GetSyncedCRDs,
		policyDir cmpath.Absolute,
		files []cmpath.Absolute,
	) ([]ast.FileObject, status.MultiError)

	// ReadClusterRegistryResources returns the list of Clusters contained in the repo.
	ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject
}
