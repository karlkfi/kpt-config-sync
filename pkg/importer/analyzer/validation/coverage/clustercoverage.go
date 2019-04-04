// Package coverage has code that helps in computing resource-cluster coverage
// relationships.
package coverage

import (
	"sort"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	sels "github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type strSet = map[string]bool

// ForCluster contains information about which clusters are covered by which cluster
// selectors.
type ForCluster struct {
	selectorNames     strSet
	selectorToCluster map[string]strSet
}

// NewForCluster creates a new cluster coverage examiner.
func NewForCluster(
	clusters []clusterregistry.Cluster,
	selectors []v1.ClusterSelector,
	errs *status.ErrorBuilder,
) *ForCluster {
	cov := ForCluster{
		selectorNames:     strSet{},
		selectorToCluster: map[string]strSet{},
	}
	for _, s := range selectors {
		cov.selectorNames[s.ObjectMeta.Name] = true
	}
	for _, s := range selectors {
		sn := s.ObjectMeta.Name
		selector, err := sels.AsPopulatedSelector(&s.Spec.Selector)
		if err != nil {
			// TODO(b/120229144): Impossible to get here.
			errs.Add(vet.InvalidSelectorError{Name: sn, Cause: err})
			continue
		}
		for _, c := range clusters {
			cn := c.ObjectMeta.Name
			if sels.IsSelected(c.ObjectMeta.Labels, selector) {
				if cov.selectorToCluster[sn] == nil {
					cov.selectorToCluster[sn] = strSet{}
				}
				cov.selectorToCluster[sn][cn] = true
			}
		}
	}
	return &cov
}

// getClusterSelectorAnnotation returns the value of the cluster selector annotation
// among the given annotations.  If the annotation is not there, "" is returned.
func getClusterSelectorAnnotation(a object.Annotated) string {
	// Looking up in a nil map will also return "".
	return a.GetAnnotations()[v1.ClusterSelectorAnnotationKey]
}

// ValidateObject validates the coverage of the object with clusters and selectors. An object
// may not have an annotation, but if it does, it has to map to a valid selector.  Also if an
// object has a selector in the annotation, that annotation must refer to a valid selector.
func (c ForCluster) ValidateObject(o *ast.FileObject, errs *status.ErrorBuilder) {
	a := getClusterSelectorAnnotation(o.MetaObject())
	if a == "" {
		return
	}
	if !c.selectorNames[a] {
		errs.Add(vet.ObjectHasUnknownClusterSelector{Resource: o, Annotation: a})
	}
}

// MapToClusters returns the names of the clusters that this object maps to.
// "" in the returned slice means "all clusters".  The output ordering is
// stable.
func (c ForCluster) MapToClusters(o metav1.Object) []string {
	a := getClusterSelectorAnnotation(o)
	if a == "" {
		return []string{""}
	}
	var cs sort.StringSlice
	for c := range c.selectorToCluster[a] {
		cs = append(cs, c)
	}
	cs.Sort()
	return cs
}
