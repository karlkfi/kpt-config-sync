package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

var (
	clusterSelectorAnnotation   = core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-selector")
	namespaceSelectorAnnotation = core.Annotation(v1.NamespaceSelectorAnnotationKey, "shipping-selector")
)

func TestClusterSelectorAnnotationValidator(t *testing.T) {
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
		nht.Pass("cluster-scoped object with both selectors",
			fake.ClusterRole(namespaceSelectorAnnotation, clusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with no annotation",
			fake.Role(),
		),
		nht.Pass("namespace-scoped object with cluster-selector",
			fake.Role(clusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with both selectors",
			fake.Role(namespaceSelectorAnnotation, clusterSelectorAnnotation),
		),
		// special cases
		nht.Pass("Cluster without annotation",
			fake.Cluster(),
		),
		nht.Pass("Kptfile",
			fake.KptFile("Kptfile", namespaceSelectorAnnotation, clusterSelectorAnnotation)),
		nht.Fail("Cluster with cluster-selector",
			fake.Cluster(clusterSelectorAnnotation),
		),
		nht.Pass("Namespace without annotation",
			fake.Namespace("namespaces/foo"),
		),
		nht.Pass("Namespace with cluster-selector",
			fake.Namespace("namespaces/foo", clusterSelectorAnnotation),
		),
		nht.Pass("ClusterSelector without annotation",
			fake.ClusterSelector(),
		),
		nht.Fail("ClusterSelector with cluster-selector",
			fake.ClusterSelector(clusterSelectorAnnotation),
		),
		nht.Pass("NamespaceSelector without annotation",
			fake.NamespaceSelector(),
		),
		nht.Fail("NamespaceSelector with cluster-selector",
			fake.NamespaceSelector(clusterSelectorAnnotation),
		),
		nht.Pass("V1Beta1 CRD without annotation",
			fake.CustomResourceDefinitionV1Beta1(),
		),
		nht.Fail("V1Beta1 CRD with cluster-selector",
			fake.CustomResourceDefinitionV1Beta1(clusterSelectorAnnotation),
		),
		nht.Pass("V1 CRD without annotation",
			fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1()),
		),
		nht.Fail("V1 CRD with cluster-selector",
			fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1(clusterSelectorAnnotation)),
		),
		// unknown types
		nht.Pass("Unknown with cluster-selector",
			fake.AnvilAtPath("", clusterSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewClusterSelectorAnnotationValidator(), testCases)
}

func TestNamespaceSelectorAnnotationValidator(t *testing.T) {
	scoper := discovery.CoreScoper()

	testCases := []nht.ValidatorTestCase{
		// Trivial Cases
		nht.Pass("empty returns no error"),
		// Scope checking
		nht.Pass("cluster-scoped object no annotation",
			fake.ClusterRole(),
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
		nht.Fail("Cluster with namespace-selector",
			fake.Cluster(namespaceSelectorAnnotation),
		),
		nht.Pass("Namespace without annotation",
			fake.Namespace("namespaces/foo"),
		),
		nht.Fail("Namespace with namespace-selector",
			fake.Namespace("namespaces/foo", namespaceSelectorAnnotation),
		),
		nht.Pass("ClusterSelector without annotation",
			fake.ClusterSelector(),
		),
		nht.Fail("ClusterSelector with namespace-selector",
			fake.ClusterSelector(namespaceSelectorAnnotation),
		),
		nht.Pass("NamespaceSelector without annotation",
			fake.NamespaceSelector(),
		),
		nht.Fail("NamespaceSelector with namespace-selector",
			fake.NamespaceSelector(namespaceSelectorAnnotation),
		),
		nht.Pass("V1Beta1 CRD without annotation",
			fake.CustomResourceDefinitionV1Beta1(),
		),
		nht.Fail("V1Beta1 CRD with namespace-selector",
			fake.CustomResourceDefinitionV1Beta1(namespaceSelectorAnnotation),
		),
		nht.Pass("V1 CRD without annotation",
			fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1()),
		),
		nht.Fail("V1 CRD with namespace-selector",
			fake.ToCustomResourceDefinitionV1(fake.CustomResourceDefinitionV1Beta1(namespaceSelectorAnnotation)),
		),
		// unknown types
		nht.Fail("Unknown with namespace-selector",
			fake.AnvilAtPath("", namespaceSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, true), testCases)
}

func TestNamespaceSelectorAnnotationValidatorServerless(t *testing.T) {
	scoper := discovery.CoreScoper()

	testCases := []nht.ValidatorTestCase{
		// Trivial Cases
		nht.Pass("empty returns no error"),
		// Scope checking
		nht.Pass("cluster-scoped object no annotation",
			fake.ClusterRole(),
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
		nht.Fail("Cluster with namespace-selector",
			fake.Cluster(namespaceSelectorAnnotation),
		),
		nht.Pass("Namespace without annotation",
			fake.Namespace("namespaces/foo"),
		),
		nht.Fail("Namespace with namespace-selector",
			fake.Namespace("namespaces/foo", namespaceSelectorAnnotation),
		),
		nht.Pass("ClusterSelector without annotation",
			fake.ClusterSelector(),
		),
		nht.Fail("ClusterSelector with namespace-selector",
			fake.ClusterSelector(namespaceSelectorAnnotation),
		),
		nht.Pass("NamespaceSelector without annotation",
			fake.NamespaceSelector(),
		),
		nht.Fail("NamespaceSelector with namespace-selector",
			fake.NamespaceSelector(namespaceSelectorAnnotation),
		),
		nht.Fail("CRD with namespace-selector",
			fake.CustomResourceDefinitionV1Beta1(namespaceSelectorAnnotation),
		),
		// unknown types
		nht.Pass("Unknown with namespace-selector",
			fake.AnvilAtPath("", namespaceSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, false), testCases)
}
