package selectors

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clustersKey struct{}
type selectorsKey struct{}

// ClusterSelectorAdder stores the clusters and selectors in the ast.Root accepting this Visitor.
type ClusterSelectorAdder struct {
	*visitor.Base

	clusters  []clusterregistry.Cluster
	selectors []v1.ClusterSelector

	errs status.MultiError
}

var _ ast.Visitor = &ClusterSelectorAdder{}

// GetClusters retrieves the selectors stored in Root.Data.
func GetClusters(r *ast.Root) ([]clusterregistry.Cluster, status.Error) {
	c, err := ast.Get(r.Data, clustersKey{})
	if err != nil {
		return nil, err
	}
	return c.([]clusterregistry.Cluster), nil
}

// GetSelectors retrieves the selectors stored in Root.Data.
func GetSelectors(r *ast.Root) ([]v1.ClusterSelector, status.Error) {
	s, err := ast.Get(r.Data, selectorsKey{})
	if err != nil {
		return nil, err
	}
	return s.([]v1.ClusterSelector), nil
}

// NewClusterSelectorAdder initializes a ClusterSelectorAdder.
func NewClusterSelectorAdder() *ClusterSelectorAdder {
	v := &ClusterSelectorAdder{
		Base: visitor.NewBase(),
	}
	v.SetImpl(v)
	return v
}

// VisitRoot stores the clusters and selectors in Root.Data.
func (v *ClusterSelectorAdder) VisitRoot(r *ast.Root) *ast.Root {
	v.clusters = getClusters(r.ClusterRegistryObjects)
	v.selectors = getSelectors(r.ClusterRegistryObjects)
	v.Base.VisitRoot(r)

	var err error
	r.Data, err = ast.Add(r.Data, clustersKey{}, v.clusters)
	v.errs = status.Append(v.errs, err)
	r.Data, err = ast.Add(r.Data, selectorsKey{}, v.selectors)
	v.errs = status.Append(v.errs, err)

	cs, err := NewClusterSelectors(v.clusters, v.selectors, r.ClusterName)
	v.errs = status.Append(v.errs, err)
	err = SetClusterSelector(cs, r)
	v.errs = status.Append(v.errs, err)

	return r
}

func (v *ClusterSelectorAdder) Error() status.MultiError {
	return v.errs
}

// FilterClusters returns the list of Clusters in the passed array of FileObjects.
func FilterClusters(objects []ast.FileObject) []clusterregistry.Cluster {
	var clusters []clusterregistry.Cluster
	for _, object := range objects {
		if o, ok := object.Object.(*clusterregistry.Cluster); ok {
			clusters = append(clusters, *o)
		}
	}
	return clusters
}

func getClusters(objects []*ast.ClusterRegistryObject) []clusterregistry.Cluster {
	objs := make([]ast.FileObject, len(objects))
	for i, obj := range objects {
		objs[i] = obj.FileObject
	}
	return FilterClusters(objs)
}

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out.
func getSelectors(objects []*ast.ClusterRegistryObject) []v1.ClusterSelector {
	var selectors []v1.ClusterSelector
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1.ClusterSelector:
			selectors = append(selectors, *o)
		}
	}
	return selectors
}
