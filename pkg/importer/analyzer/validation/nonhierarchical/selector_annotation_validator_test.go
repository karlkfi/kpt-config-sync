package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

func TestSelectorAnnotationValidator(t *testing.T) {
	scoper := discovery.Scoper{
		kinds.Role().GroupKind():        discovery.NamespaceScope,
		kinds.ClusterRole().GroupKind(): discovery.ClusterScope,

		// The logic doesn't make a special case for these types, but this is the scope
		// the APIServer will report.
		kinds.Namespace().GroupKind():         discovery.ClusterScope,
		kinds.Cluster().GroupKind():           discovery.ClusterScope,
		kinds.ClusterSelector().GroupKind():   discovery.ClusterScope,
		kinds.NamespaceSelector().GroupKind(): discovery.ClusterScope,
	}

	clusterSelectorAnnotation := core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-selector")
	namespaceSelectorAnnotation := core.Annotation(v1.NamespaceSelectorAnnotationKey, "shipping-selector")

	testCases := []nht.ValidatorTestCase{
		// Trivial Cases
		nht.Pass("empty returns no error"),
		// Scope checking
		nht.Pass("cluster-scoped object no annotation",
			fake.ClusterRole(),
		),
		nht.Pass("cluster-scoped object with cluster-selector",
			fake.ClusterRole(clusterSelectorAnnotation),
		),
		nht.Fail("cluster-scoped object with namespace-selector",
			fake.ClusterRole(namespaceSelectorAnnotation),
		),
		nht.Fail("cluster-scoped object with both selectors",
			fake.ClusterRole(namespaceSelectorAnnotation, clusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with no annotation",
			fake.Role(),
		),
		nht.Pass("namespace-scoped object with cluster-selector",
			fake.Role(clusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with namespace-selector",
			fake.Role(namespaceSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with both selectors",
			fake.Role(namespaceSelectorAnnotation, clusterSelectorAnnotation),
		),
		// special cases
		nht.Pass("Cluster without annotation",
			fake.Cluster(),
		),
		nht.Fail("Cluster with cluster-selector",
			fake.Cluster(clusterSelectorAnnotation),
		),
		nht.Fail("Cluster with namespace-selector",
			fake.Cluster(namespaceSelectorAnnotation),
		),
		nht.Pass("Namespace without annotation",
			fake.Namespace("namespaces/foo"),
		),
		nht.Pass("Namespace with cluster-selector",
			fake.Namespace("namespaces/foo", clusterSelectorAnnotation),
		),
		nht.Fail("Namespace with namespace-selector",
			fake.Namespace("namespaces/foo", namespaceSelectorAnnotation),
		),
		nht.Pass("ClusterSelector without annotation",
			fake.ClusterSelector(),
		),
		nht.Fail("ClusterSelector with cluster-selector",
			fake.ClusterSelector(clusterSelectorAnnotation),
		),
		nht.Fail("ClusterSelector with namespace-selector",
			fake.ClusterSelector(namespaceSelectorAnnotation),
		),
		nht.Pass("NamespaceSelector without annotation",
			fake.NamespaceSelector(),
		),
		nht.Fail("NamespaceSelector with cluster-selector",
			fake.NamespaceSelector(clusterSelectorAnnotation),
		),
		nht.Fail("NamespaceSelector with namespace-selector",
			fake.NamespaceSelector(namespaceSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewSelectorAnnotationValidator(scoper), testCases)
}
