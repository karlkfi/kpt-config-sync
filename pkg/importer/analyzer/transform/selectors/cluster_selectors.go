package selectors

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FilterClusters returns the list of Clusters in the passed array of FileObjects.
func FilterClusters(objects []ast.FileObject) ([]clusterregistry.Cluster, status.MultiError) {
	var clusters []clusterregistry.Cluster
	var errs status.MultiError
	for _, object := range objects {
		if object.GetObjectKind().GroupVersionKind() != kinds.Cluster() {
			continue
		}
		if s, err := object.Structured(); err != nil {
			errs = status.Append(errs, err)
		} else {
			o := s.(*clusterregistry.Cluster)
			clusters = append(clusters, *o)
		}
	}
	return clusters, errs
}

// ObjectHasUnknownClusterSelector reports that `resource`'s cluster-selector annotation
// references a ClusterSelector that does not exist.
func ObjectHasUnknownClusterSelector(resource client.Object, annotation string) status.Error {
	return objectHasUnknownSelector.
		Sprintf("Config %q MUST refer to an existing ClusterSelector, but has annotation \"%s=%s\" which maps to no declared ClusterSelector",
			resource.GetName(), v1.LegacyClusterSelectorAnnotationKey, annotation).
		BuildWithResources(resource)
}
