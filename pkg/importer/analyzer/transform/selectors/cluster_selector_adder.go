package selectors

import (
	"os"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
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

	errs status.ErrorBuilder
}

var _ ast.Visitor = &ClusterSelectorAdder{}

// GetClusters retrieves the selectors stored in Root.Data.
func GetClusters(r *ast.Root) []clusterregistry.Cluster {
	return r.Data.Get(clustersKey{}).([]clusterregistry.Cluster)
}

// GetSelectors retrieves the selectors stored in Root.Data.
func GetSelectors(r *ast.Root) []v1.ClusterSelector {
	return r.Data.Get(selectorsKey{}).([]v1.ClusterSelector)
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

	r.Data = r.Data.Add(clustersKey{}, v.clusters)
	r.Data = r.Data.Add(selectorsKey{}, v.selectors)

	cs, err := NewClusterSelectors(v.clusters, v.selectors, os.Getenv("CLUSTER_NAME"))
	v.errs.Add(err)
	SetClusterSelector(cs, r)

	return r
}

func (v *ClusterSelectorAdder) Error() *status.MultiError {
	return v.errs.Build()
}

func getClusters(objects []*ast.ClusterRegistryObject) []clusterregistry.Cluster {
	var clusters []clusterregistry.Cluster
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *clusterregistry.Cluster:
			clusters = append(clusters, *o)
		}
	}
	return clusters
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
