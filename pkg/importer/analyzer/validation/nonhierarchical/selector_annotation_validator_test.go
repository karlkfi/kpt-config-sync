package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
)

var (
	legacyClusterSelectorAnnotation = core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "prod-selector")
	inlineClusterSelectorAnnotation = core.Annotation(v1alpha1.ClusterNameSelectorAnnotationKey, "prod-cluster")
	namespaceSelectorAnnotation     = core.Annotation(v1.NamespaceSelectorAnnotationKey, "shipping-selector")
)

func TestClusterSelectorAnnotationValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		// Trivial Cases
		nht.Pass("empty returns no error"),
		// Scope checking
		nht.Pass("cluster-scoped object no annotation",
			fake.ClusterRole(),
		),
		nht.Pass("cluster-scoped object with legacy cluster-selector",
			fake.ClusterRole(legacyClusterSelectorAnnotation),
		),
		nht.Pass("cluster-scoped object with inline cluster-selector",
			fake.ClusterRole(inlineClusterSelectorAnnotation),
		),
		nht.Pass("cluster-scoped object with both namespace selector and legacy cluster-selector",
			fake.ClusterRole(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
		),
		nht.Pass("cluster-scoped object with both namespace selector and inline cluster-selector",
			fake.ClusterRole(namespaceSelectorAnnotation, inlineClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with no annotation",
			fake.Role(),
		),
		nht.Pass("namespace-scoped object with legacy cluster-selector",
			fake.Role(legacyClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with inline cluster-selector",
			fake.Role(inlineClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with namespace selector and legacy cluster-selector",
			fake.Role(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with namespace selector and inline cluster-selector",
			fake.Role(namespaceSelectorAnnotation, inlineClusterSelectorAnnotation),
		),
		// special cases
		nht.Pass("Cluster without annotation",
			fake.Cluster(),
		),
		nht.Pass("Kptfile",
			fake.KptFile("Kptfile", namespaceSelectorAnnotation, legacyClusterSelectorAnnotation)),
		nht.Fail("Cluster with legacy cluster-selector",
			fake.Cluster(legacyClusterSelectorAnnotation),
		),
		nht.Fail("Cluster with inline cluster-selector",
			fake.Cluster(inlineClusterSelectorAnnotation),
		),
		nht.Pass("Namespace without annotation",
			fake.Namespace("namespaces/foo"),
		),
		nht.Pass("Namespace with legacy cluster-selector",
			fake.Namespace("namespaces/foo", legacyClusterSelectorAnnotation),
		),
		nht.Pass("Namespace with inline cluster-selector",
			fake.Namespace("namespaces/foo", inlineClusterSelectorAnnotation),
		),
		nht.Pass("ClusterSelector without annotation",
			fake.ClusterSelector(),
		),
		nht.Fail("ClusterSelector with legacy cluster-selector",
			fake.ClusterSelector(legacyClusterSelectorAnnotation),
		),
		nht.Fail("ClusterSelector with inline cluster-selector",
			fake.ClusterSelector(inlineClusterSelectorAnnotation),
		),
		nht.Pass("NamespaceSelector without annotation",
			fake.NamespaceSelector(),
		),
		nht.Fail("NamespaceSelector with legacy cluster-selector",
			fake.NamespaceSelector(legacyClusterSelectorAnnotation),
		),
		nht.Fail("NamespaceSelector with inline cluster-selector",
			fake.NamespaceSelector(inlineClusterSelectorAnnotation),
		),
		nht.Pass("V1Beta1 CRD without annotation",
			fake.CustomResourceDefinitionV1Beta1(),
		),
		nht.Fail("V1Beta1 CRD with legacy cluster-selector",
			fake.CustomResourceDefinitionV1Beta1(legacyClusterSelectorAnnotation),
		),
		nht.Fail("V1Beta1 CRD with inline cluster-selector",
			fake.CustomResourceDefinitionV1Beta1(inlineClusterSelectorAnnotation),
		),
		nht.Pass("V1 CRD without annotation",
			fake.CustomResourceDefinitionV1(),
		),
		nht.Fail("V1 CRD with legacy cluster-selector",
			fake.CustomResourceDefinitionV1(legacyClusterSelectorAnnotation),
		),
		nht.Fail("V1 CRD with inline cluster-selector",
			fake.CustomResourceDefinitionV1(inlineClusterSelectorAnnotation),
		),
		// unknown types
		nht.Pass("Unknown with legacy cluster-selector",
			fake.AnvilAtPath("", legacyClusterSelectorAnnotation),
		),
		nht.Pass("Unknown with cluster-selector",
			fake.AnvilAtPath("", inlineClusterSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewClusterSelectorAnnotationValidator(), testCases)
}

func TestNamespaceSelectorAnnotationValidator(t *testing.T) {
	scoper := discovery.CoreScoper(true)

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
			fake.ClusterRole(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with no annotation",
			fake.Role(),
		),
		nht.Pass("namespace-scoped object with namespace-selector",
			fake.Role(namespaceSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with both selectors",
			fake.Role(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
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
			fake.CustomResourceDefinitionV1(),
		),
		nht.Fail("V1 CRD with namespace-selector",
			fake.CustomResourceDefinitionV1(namespaceSelectorAnnotation),
		),
		// unknown types
		nht.Fail("Unknown with namespace-selector",
			fake.AnvilAtPath("", namespaceSelectorAnnotation),
		),
	}

	nht.RunAll(t, nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, true), testCases)
}

func TestNamespaceSelectorAnnotationValidatorServerless(t *testing.T) {
	scoper := discovery.CoreScoper(true)

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
			fake.ClusterRole(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with no annotation",
			fake.Role(),
		),
		nht.Pass("namespace-scoped object with namespace-selector",
			fake.Role(namespaceSelectorAnnotation),
		),
		nht.Pass("namespace-scoped object with both selectors",
			fake.Role(namespaceSelectorAnnotation, legacyClusterSelectorAnnotation),
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
