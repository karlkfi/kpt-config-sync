package filesystem

// Implements the operations needed to enforce the /clusterregistry
// directory structure.

import (
	policyhierarchy "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out. dirname is the directory
// name relative to the root directory of the repository, and infos is the set
// of resource data that were read from the directory.
func processClusterRegistryDir(dirname string, infos []*resource.Info) ([]clusterregistry.Cluster, []policyhierarchy.ClusterSelector, error) {
	v := newValidator()
	var crc []clusterregistry.Cluster
	var css []policyhierarchy.ClusterSelector
	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, dirname)
		// Cluster and ClusterSelector types are allowed in this directory, and
		// nothing else.
		gvk := o.GetObjectKind().GroupVersionKind()
		switch gvk {
		case policyhierarchy.SchemeGroupVersion.WithKind("ClusterSelector"):
			var cs policyhierarchy.ClusterSelector
			if err := convertUnstructured(o, &cs, i.Source); err != nil {
				return nil, nil, err
			}
			css = append(css, cs)
		case clusterregistry.SchemeGroupVersion.WithKind("Cluster"):
			var c clusterregistry.Cluster
			if err := convertUnstructured(o, &c, i.Source); err != nil {
				return nil, nil, err
			}
			crc = append(crc, c)
		default:
			// No other objects are allowed in the clusterregistry directory.
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
	}
	return crc, css, v.err
}
