package selectors

import (
	"os"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clustersKey struct{}
type selectorsKey struct{}

// ClusterSelectorAdder stores the clusters and selectors in the ast.Root accepting this Visitor.
type ClusterSelectorAdder struct {
	*visitor.Base

	clusters  []clusterregistry.Cluster
	selectors []v1alpha1.ClusterSelector

	errs multierror.Builder
}

// GetClusters retrieves the selectors stored in Root.Data.
func GetClusters(r *ast.Root) []clusterregistry.Cluster {
	return r.Data.Get(clustersKey{}).([]clusterregistry.Cluster)
}

// GetSelectors retrieves the selectors stored in Root.Data.
func GetSelectors(r *ast.Root) []v1alpha1.ClusterSelector {
	return r.Data.Get(selectorsKey{}).([]v1alpha1.ClusterSelector)
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
	v.Base.VisitRoot(r)

	r.Data = r.Data.Add(clustersKey{}, v.clusters)
	r.Data = r.Data.Add(selectorsKey{}, v.selectors)

	cs, err := NewClusterSelectors(v.clusters, v.selectors, os.Getenv("CLUSTER_NAME"))
	// TODO(b/120229144): To be factored into KNV Error.
	v.errs.Add(errors.Wrapf(err, "could not create cluster selectors"))
	SetClusterSelector(cs, r)

	return r
}

// VisitClusterRegistry records the clusters and selectors in clusterregistry/.
func (v *ClusterSelectorAdder) VisitClusterRegistry(c *ast.ClusterRegistry) *ast.ClusterRegistry {
	v.clusters = getClusters(c.Objects)
	v.selectors = getSelectors(c.Objects)

	return v.Base.VisitClusterRegistry(c)
}

func (v *ClusterSelectorAdder) Error() error {
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
func getSelectors(objects []*ast.ClusterRegistryObject) []v1alpha1.ClusterSelector {
	var selectors []v1alpha1.ClusterSelector
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.ClusterSelector:
			selectors = append(selectors, *o)
		}
	}
	return selectors
}
