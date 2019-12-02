package filesystem_test

import (
	"encoding/json"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/testing/parsertest"
	"github.com/google/nomos/testing/testoutput"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

var engineerGVK = schema.GroupVersionKind{
	Group:   "employees",
	Version: "v1alpha1",
	Kind:    "Engineer",
}

func engineerCRD(opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	obj := fake.CustomResourceDefinitionObject(opts...)
	obj.Name = "engineers.employees"
	obj.Spec.Group = "employees"
	obj.Spec.Scope = v1beta1.NamespaceScoped
	obj.Spec.Names = v1beta1.CustomResourceDefinitionNames{
		Plural:   "engineers",
		Singular: "engineer",
		Kind:     "Engineer",
	}
	obj.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{
		{
			Name:    "v1alpha1",
			Served:  true,
			Storage: true,
		},
	}
	return obj
}

func scopedResourceQuota(path string, opts ...core.MetaMutator) ast.FileObject {
	obj := fake.ResourceQuotaObject(opts...)
	obj.Spec.Scopes = []corev1.ResourceQuotaScope{"Terminating"}
	obj.Spec.ScopeSelector = &corev1.ScopeSelector{
		MatchExpressions: []corev1.ScopedResourceSelectorRequirement{
			{Operator: "In", ScopeName: "PriorityClass"},
		},
	}
	return fake.FileObject(obj, path)
}

func TestParseRepo(t *testing.T) {
	test := parsertest.Test{
		NewParser: parsertest.NewParser,
		TestCases: []parsertest.TestCase{
			parsertest.Failure("missing repo",
				system.MissingRepoErrorCode),
			parsertest.Failure("invalid repo version",
				system.UnsupportedRepoSpecVersionCode,
				fake.Repo(fake.RepoVersion("0.0.0"))),
		},
	}

	test.RunAll(t)
}

func TestParserVetErrors(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Success("ResourceQuota with scope and normal quota",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.Source("namespaces/bar/namespace.yaml")),
				scopedResourceQuota("namespaces/bar/rq.yaml",
					testoutput.Source("namespaces/bar/rq.yaml"),
					core.Namespace("bar"),
				),
			),
			fake.Namespace("namespaces/bar"),
			scopedResourceQuota("namespaces/bar/rq.yaml"),
		),
		parsertest.Failures("ResourceQuota with scope and hierarchical quota",
			[]string{validation.IllegalResourceQuotaFieldErrorCode, validation.IllegalResourceQuotaFieldErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeHierarchicalQuota,
				kinds.ResourceQuota().GroupVersion(), kinds.ResourceQuota().Kind)),
			fake.Namespace("namespaces/bar"),
			scopedResourceQuota("namespaces/bar/rq.yaml"),
		),
		parsertest.Success("Engineer CustomResourceDefinition",
			testoutput.NewAllConfigs(t,
				fake.FileObject(engineerCRD(testoutput.Source("cluster/engineer-crd.yaml")),
					"cluster/engineer-crd.yaml"),
			),
			fake.FileObject(engineerCRD(), "cluster/engineer-crd.yaml"),
		),
		parsertest.Success("Engineer CustomResourceDefinition and CustomResource",
			testoutput.NewAllConfigsWithCRDs(t, []*restmapper.APIGroupResources{engineerResource},
				fake.FileObject(engineerCRD(testoutput.Source("cluster/engineer-crd.yaml")),
					"cluster/engineer-crd.yaml"),
				fake.Namespace("namespaces/bar", testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.FileObject(fake.UnstructuredObject(engineerGVK,
					core.Namespace("bar"), testoutput.Source("namespaces/bar/engineer.yaml"),
				), "namespaces/bar/engineer.yaml"),
			),
			fake.FileObject(engineerCRD(), "cluster/engineer-crd.yaml"),
			fake.Namespace("namespaces/bar"),
			fake.FileObject(fake.UnstructuredObject(engineerGVK), "namespaces/bar/engineer.yaml"),
		),
		parsertest.Failure("Engineer CustomResource without CRD",
			validation.UnknownObjectErrorCode,
			fake.Namespace("namespaces/bar"),
			fake.FileObject(fake.UnstructuredObject(engineerGVK), "namespaces/bar/engineer.yaml"),
		),
		parsertest.Success("Valid to have Abstract Namespace with both Namespace and Abstract Namespace children",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar/foo", testoutput.Source("namespaces/bar/foo/namespace.yaml")),
				fake.Namespace("namespaces/bar/qux/lym", testoutput.Source("namespaces/bar/qux/lym/namespace.yaml")),
			),
			fake.Namespace("namespaces/bar/foo"),
			fake.Namespace("namespaces/bar/qux/lym")),
		parsertest.Failures("Namespace dir with Abstract Namespace child",
			[]string{
				validation.IllegalNamespaceSubdirectoryErrorCode,
				semantic.UnsyncableResourcesErrorCode,
			},
			fake.Namespace("namespaces/bar"),
			fake.RoleAtPath("namespaces/bar/foo/rb.yaml"),
		),
		parsertest.Failure("Unsyncable resources because never instantiated",
			semantic.UnsyncableResourcesErrorCode,
			fake.RoleBindingAtPath("namespaces/rb.yaml"),
		),
		parsertest.Failure("Abstract Namespace dir with uninheritable Rolebinding",
			validation.IllegalAbstractNamespaceObjectKindErrorCode,
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeNone,
				kinds.RoleBinding().GroupVersion(), kinds.RoleBinding().Kind)),
			fake.RoleBindingAtPath("namespaces/rb.yaml"),
			fake.Namespace("namespaces/bar"),
		),
		parsertest.Success("NamespaceSelector",
			testoutput.NewAllConfigs(t,
				fake.RoleBindingAtPath("namespaces/bar/rb.yaml", core.Name("sre"),
					inlinedNamespaceSelectorAnnotation(t, namespaceSelector("sre-supported", "env", "prod")),
					core.Namespace("prod-ns"),
					testoutput.Source("namespaces/bar/rb.yaml"),
				),
				fake.Namespace("namespaces/bar/prod-ns",
					core.Label("env", "prod"),
					testoutput.Source("namespaces/bar/prod-ns/namespace.yaml"),
				),
				fake.Namespace("namespaces/bar/test-ns",
					core.Label("env", "test"),
					testoutput.Source("namespaces/bar/test-ns/namespace.yaml"),
				),
			),
			fake.FileObject(namespaceSelector("sre-supported", "env", "prod"),
				"namespaces/bar/ns-selector.yaml"),
			fake.RoleBindingAtPath("namespaces/bar/rb.yaml", core.Name("sre"),
				namespaceSelectorAnnotation("sre-supported")),
			fake.Namespace("namespaces/bar/prod-ns", core.Label("env", "prod")),
			fake.Namespace("namespaces/bar/test-ns", core.Label("env", "test")),
		),
		parsertest.Success("minimal repo",
			testoutput.NewAllConfigs(t),
		),
		parsertest.Success("Multiple resources with HierarchyConfigs",
			testoutput.NewAllConfigs(t),
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.ResourceQuota().GroupVersion(), kinds.ResourceQuota().Kind)),
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.Role().GroupVersion(), kinds.Role().Kind)),
		),
		parsertest.Failure("Namespaces dir with Namespace",
			metadata.IllegalTopLevelNamespaceErrorCode,
			fake.Namespace("namespaces")),
		parsertest.Success("Namespaces dir with ResourceQuota and hierarchical quota inheritance",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.FileObject(resourceQuotaObject(core.Name("config-management-resource-quota"),
					core.Namespace("bar"),
					core.Label("nomos-namespace-type", "workload"),
					testoutput.Source("namespaces/rq.yaml"),
				), "namespaces/rq.yaml"),
				fake.FileObject(fake.HierarchicalQuotaObject(
					fake.HierarchicalQuotaRoot(
						fake.HierarchicalQuotaNode("namespaces", v1.HierarchyNodeAbstractNamespace,
							resourceQuotaObject(core.Name("config-management-resource-quota"),
								testoutput.Source("namespaces/rq.yaml")),
							fake.HierarchicalQuotaNode("bar", v1.HierarchyNodeNamespace,
								resourceQuotaObject(core.Name("config-management-resource-quota"),
									core.Label("nomos-namespace-type", "workload"),
									testoutput.Source("namespaces/rq.yaml")),
							),
						),
					),
				), ""),
			),
			fake.HierarchyConfigAtPath("system/rq.yaml", core.Name("resourcequota"),
				fake.HierarchyConfigResource(v1.HierarchyModeHierarchicalQuota,
					kinds.ResourceQuota().GroupVersion(), kinds.ResourceQuota().Kind)),
			fake.FileObject(resourceQuotaObject(core.Name("pod-quota")), "namespaces/rq.yaml"),
			fake.Namespace("namespaces/bar"),
		),
		parsertest.Success("Namespace with multiple inherited RoleBindings",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/foo", testoutput.Source("namespaces/foo/namespace.yaml")),
				fake.RoleBinding(core.Name("alice"), core.Namespace("foo"),
					testoutput.Source("namespaces/rb-1.yaml")),
				fake.RoleBinding(core.Name("bob"), core.Namespace("foo"),
					testoutput.Source("namespaces/rb-2.yaml")),
			),
			fake.RoleBindingAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.RoleBindingAtPath("namespaces/rb-2.yaml", core.Name("bob")),
			fake.Namespace("namespaces/foo")),
		parsertest.Failure("Cluster-scoped objects must not collide",
			nonhierarchical.NameCollisionErrorCode,
			fake.ClusterRoleAtPath("cluster/cr-1.yaml", core.Name("alice")),
			fake.ClusterRoleAtPath("cluster/cr-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Namespaces must be uniquely named",
			nonhierarchical.NameCollisionErrorCode,
			fake.Namespace("namespaces/foo/bar"),
			fake.Namespace("namespaces/qux/bar"),
		),
		parsertest.Success("Two abstract Namespace dirs with non-unique names are allowed.",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/foo/foo/bar", testoutput.Source("namespaces/foo/foo/bar/namespace.yaml")),
			),
			fake.Namespace("namespaces/foo/foo/bar"),
		),
		parsertest.Success("An abstract namespace and a leaf namespace may share a name",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar/foo", testoutput.Source("namespaces/bar/foo/namespace.yaml")),
				fake.Namespace("namespaces/foo/bar", testoutput.Source("namespaces/foo/bar/namespace.yaml")),
			),
			fake.Namespace("namespaces/bar/foo"),
			fake.Namespace("namespaces/foo/bar"),
		),
		parsertest.Success("kube-* is a system dir but is allowed",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/kube-something",
					testoutput.Source("namespaces/kube-something/namespace.yaml")),
			),
			fake.Namespace("namespaces/kube-something"),
		),
		parsertest.Success("kube-system is a system dir but is allowed",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/kube-system", testoutput.Source("namespaces/kube-system/namespace.yaml")),
			),
			fake.Namespace("namespaces/kube-system"),
		),
		parsertest.Success("Default namespace is allowed",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/default", testoutput.Source("namespaces/default/namespace.yaml")),
			),
			fake.Namespace("namespaces/default"),
		),
		parsertest.Failure("Dir name invalid",
			syntax.InvalidDirectoryNameErrorCode,
			fake.Namespace("namespaces/foo bar"),
		),
		parsertest.Failure("Namespace with NamespaceSelector annotation is invalid",
			metadata.IllegalNamespaceAnnotationErrorCode,
			fake.Namespace("namespaces/bar", core.Annotation(v1.NamespaceSelectorAnnotationKey, "prod")),
		),
		parsertest.Failure("NamespaceSelector may not have clusterSelector annotations",
			validation.NamespaceSelectorMayNotHaveAnnotationCode,
			fake.FileObject(clusterSelectorObject("prod-cluster", "env", "prod"),
				"clusterregistry/cs.yaml"),
			fake.NamespaceSelector(clusterSelectorAnnotation("prod-cluster")),
		),
		parsertest.Failure("Namespace-scoped object in cluster/ dir",
			validation.IllegalKindInClusterErrorCode,
			fake.RoleBindingAtPath("cluster/rb.yaml"),
		),
		parsertest.Failure("Illegal annotation definition is an error",
			metadata.IllegalAnnotationDefinitionErrorCode,
			fake.ClusterRole(core.Annotation("configmanagement.gke.io/stuff", "prod")),
		),
		parsertest.Failure("Illegal label definition is an error",
			metadata.IllegalLabelDefinitionErrorCode,
			fake.ClusterRole(core.Label("configmanagement.gke.io/stuff", "prod")),
		),
		parsertest.Failure("Illegal object declaration in system/ is an error",
			system.IllegalKindInSystemErrorCode,
			fake.RoleAtPath("system/role.yaml"),
		),
		parsertest.Failure("Duplicate Repo definitions is an error",
			semantic.MultipleSingletonsErrorCode,
			fake.Repo(),
			fake.Repo(),
		),
		parsertest.Failure("Name collision in namespace",
			nonhierarchical.NameCollisionErrorCode,
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/foo/role-1.yaml", core.Name("alice")),
			fake.RoleAtPath("namespaces/foo/role-2.yaml", core.Name("alice")),
		),
		parsertest.Success("No name collision if different types",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/foo", testoutput.Source("namespaces/foo/namespace.yaml")),
				fake.Role(core.Name("alice"), core.Namespace("foo"),
					testoutput.Source("namespaces/foo/role.yaml")),
				fake.RoleBinding(core.Name("alice"), core.Namespace("foo"),
					testoutput.Source("namespaces/foo/rolebinding.yaml")),
			),
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/foo/role.yaml", core.Name("alice")),
			fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml", core.Name("alice")),
		),
		parsertest.Failure("Name collision in child node",
			nonhierarchical.NameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleAtPath("namespaces/foo/bar/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Name collision in grandchild node",
			nonhierarchical.NameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/bar/qux"),
			fake.RoleAtPath("namespaces/foo/bar/qux/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Success("No name collision in sibling nodes",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/foo/bar", testoutput.Source("namespaces/foo/bar/namespace.yaml")),
				fake.RoleBinding(core.Name("alice"), core.Namespace("bar"),
					testoutput.Source("namespaces/foo/bar/rb-1.yaml")),
				fake.Namespace("namespaces/foo/qux", testoutput.Source("namespaces/foo/qux/namespace.yaml")),
				fake.RoleBinding(core.Name("alice"), core.Namespace("qux"),
					testoutput.Source("namespaces/foo/qux/rb-2.yaml")),
			),
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleBindingAtPath("namespaces/foo/bar/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/qux"),
			fake.RoleBindingAtPath("namespaces/foo/qux/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Empty string name is an error",
			nonhierarchical.MissingObjectNameErrorCode,
			fake.ClusterRole(core.Name(""))),
		parsertest.Failure("Specifying resourceVersion is an error",
			syntax.IllegalFieldsInConfigErrorCode,
			fake.ClusterRole(resourceVersion("999")),
		),
		parsertest.Failures("Repo outside system/ is an error",
			[]string{
				syntax.IllegalSystemResourcePlacementErrorCode,
				nonhierarchical.UnsupportedObjectErrorCode,
			},
			fake.RepoAtPath("cluster/repo.yaml")),
		parsertest.Failures("HierarchyConfig outside system/ is an error",
			[]string{
				syntax.IllegalSystemResourcePlacementErrorCode,
				nonhierarchical.UnsupportedObjectErrorCode,
			},
			fake.HierarchyConfigAtPath("cluster/hc.yaml")),
		parsertest.Failures("HierarchyConfig contains a CRD",
			[]string{
				hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode,
				hierarchyconfig.ClusterScopedResourceInHierarchyConfigErrorCode,
			},
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.NamespaceConfig().GroupVersion(), kinds.NamespaceConfig().Kind)),
		),
		parsertest.Failures("HierarchyConfig contains a Namespace",
			[]string{
				hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode,
				hierarchyconfig.ClusterScopedResourceInHierarchyConfigErrorCode,
			},
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.NamespaceConfig().GroupVersion(), kinds.NamespaceConfig().Kind)),
		),
		parsertest.Failures("HierarchyConfig contains a NamespaceConfig",
			[]string{
				hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode,
				hierarchyconfig.ClusterScopedResourceInHierarchyConfigErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.NamespaceConfig().GroupVersion(), kinds.NamespaceConfig().Kind)),
		),
		parsertest.Failures("HierarchyConfig contains a Sync",
			[]string{
				hierarchyconfig.UnsupportedResourceInHierarchyConfigErrorCode,
				hierarchyconfig.ClusterScopedResourceInHierarchyConfigErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.Sync().GroupVersion(), kinds.Sync().Kind)),
		),
		parsertest.Failure("Invalid name for ClusterRole",
			nonhierarchical.InvalidMetadataNameErrorCode,
			fake.ClusterRole(core.Name("RBAC")),
		),
		parsertest.Failure("Illegal Namespace in clusterregistry/",
			syntax.IllegalKindInClusterregistryErrorCode,
			fake.Namespace("clusterregistry"),
		),
		parsertest.Failure("Illegal NamespaceSelector in Namespace directory.",
			syntax.IllegalKindInNamespacesErrorCode,
			fake.Namespace("namespaces/foo"),
			fake.FileObject(namespaceSelectorObject("sel", "env", "prod"), "namespaces/foo/ns.yam"),
		),
		parsertest.Failure("Resource with UID specified",
			syntax.IllegalFieldsInConfigErrorCode,
			fake.Namespace("namespaces/foo", core.UID("illegal-uid")),
		),
	)

	test.RunAll(t)
}

func resourceQuotaObject(opts ...core.MetaMutator) *corev1.ResourceQuota {
	obj := fake.ResourceQuotaObject(opts...)
	podQ, _ := resource.ParseQuantity("10")
	obj.Spec.Hard = map[corev1.ResourceName]resource.Quantity{corev1.ResourcePods: podQ}
	return obj
}

func namespaceSelector(name, key, value string, opts ...core.MetaMutator) *v1.NamespaceSelector {
	obj := fake.NamespaceSelectorObject(opts...)
	obj.Name = name
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func namespaceSelectorAnnotation(name string) core.MetaMutator {
	return core.Annotation(v1.NamespaceSelectorAnnotationKey, name)
}

func inlinedNamespaceSelectorAnnotation(t *testing.T, selector *v1.NamespaceSelector) core.MetaMutator {
	content, err := json.Marshal(selector)
	if err != nil {
		t.Error(err)
	}
	return core.Annotation(v1.NamespaceSelectorAnnotationKey, string(content))
}

func clusterSelectorAnnotation(value string) core.MetaMutator {
	return core.Annotation(v1.ClusterSelectorAnnotationKey, value)
}

func inlinedClusterSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) core.MetaMutator {
	content, err := json.Marshal(selector)
	if err != nil {
		t.Error(err)
	}
	return core.Annotation(v1.ClusterSelectorAnnotationKey, string(content))
}

func cluster(name string, opts ...core.MetaMutator) ast.FileObject {
	mutators := append(opts, core.Name(name))
	return fake.Cluster(mutators...)
}

func namespaceSelectorObject(name, key, value string) *v1.NamespaceSelector {
	obj := fake.NamespaceSelectorObject(core.Name(name))
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func clusterSelectorObject(name, key, value string) *v1.ClusterSelector {
	obj := fake.ClusterSelectorObject(core.Name(name))
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func inlinedSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) core.MetaMutator {
	return inlinedClusterSelectorAnnotation(t, selector)
}

func resourceVersion(version string) core.MetaMutator {
	return func(meta core.Object) {
		meta.SetResourceVersion(version)
	}
}

// Test how the parser handles ClusterSelectors
func TestParseClusterSelector(t *testing.T) {
	prodCluster := "cluster-1"
	devCluster := "cluster-2"

	prodSelectorName := "sel-1"
	prodLabel := core.Label("environment", "prod")
	prodSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(prodSelectorName, "environment", "prod")
	}
	prodSelectorAnnotation := clusterSelectorAnnotation(prodSelectorName)
	prodSelectorAnnotationInlined := inlinedSelectorAnnotation(t, prodSelectorObject())

	devSelectorName := "sel-2"
	devLabel := core.Label("environment", "dev")
	devSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(devSelectorName, "environment", "dev")
	}
	devSelectorAnnotation := clusterSelectorAnnotation(devSelectorName)
	devSelectorAnnotationInlined := inlinedSelectorAnnotation(t, devSelectorObject())

	test := parsertest.VetTest(
		parsertest.Success("Resource without selector always exists 1",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(prodCluster),
					testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.RoleBinding(core.Namespace("bar"), testoutput.InCluster(prodCluster),
					testoutput.Source("namespaces/bar/rolebinding.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Resource without selector always exists 2",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(devCluster),
					testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.RoleBinding(core.Namespace("bar"), testoutput.InCluster(devCluster),
					testoutput.Source("namespaces/bar/rolebinding.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Namespace resource selected",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(prodCluster),
					testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.RoleBinding(core.Namespace("bar"), testoutput.InCluster(prodCluster),
					prodSelectorAnnotationInlined, testoutput.Source("namespaces/bar/rolebinding.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace resource not selected",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(devCluster),
					testoutput.Source("namespaces/bar/namespace.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Namespace selected",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(prodCluster), prodSelectorAnnotationInlined,
					testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.RoleBinding(core.Namespace("bar"), testoutput.InCluster(prodCluster),
					testoutput.Source("namespaces/bar/rolebinding.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace not selected",
			testoutput.NewAllConfigs(t),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Cluster resource selected",
			testoutput.NewAllConfigs(t,
				fake.ClusterRoleBinding(prodSelectorAnnotationInlined, testoutput.InCluster(prodCluster),
					testoutput.Source("cluster/crb.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Cluster resource not selected",
			testoutput.NewAllConfigs(t),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Abstract Namespace resource selected",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/foo/bar", testoutput.InCluster(prodCluster),
					testoutput.Source("namespaces/foo/bar/namespace.yaml")),
				fake.ConfigMapAtPath("", core.Namespace("bar"), prodSelectorAnnotationInlined,
					testoutput.InCluster(prodCluster), testoutput.Source("namespaces/foo/configmap.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.HierarchyConfig(fake.HierarchyConfigResource(v1.HierarchyModeInherit,
				kinds.ConfigMap().GroupVersion(), kinds.ConfigMap().Kind)),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/foo/bar"),
			fake.ConfigMapAtPath("namespaces/foo/configmap.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Colliding resources selected to different clusters may coexist",
			testoutput.NewAllConfigs(t,
				fake.Namespace("namespaces/bar", testoutput.InCluster(devCluster),
					testoutput.Source("namespaces/bar/namespace.yaml")),
				fake.RoleBinding(core.Namespace("bar"), devSelectorAnnotationInlined,
					testoutput.InCluster(devCluster), testoutput.Source("namespaces/bar/rolebinding-2.yaml")),
			),
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),
			fake.FileObject(devSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding-1.yaml", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding-2.yaml", devSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Failure(
			"A namespaced object that has a cluster selector annotation for nonexistent cluster is an error",
			coverage.ObjectHasUnknownClusterSelectorCode,
			fake.Namespace("namespaces/foo", clusterSelectorAnnotation("does-not-exist")),
		),
		parsertest.Failure(
			"A cluster object that has a cluster selector annotation for nonexistent cluster is an error",
			coverage.ObjectHasUnknownClusterSelectorCode,
			fake.ClusterRole(clusterSelectorAnnotation("does-not-exist")),
		),
		parsertest.Success("A subdir of cluster/ is ok",
			testoutput.NewAllConfigs(t,
				fake.ClusterRoleBinding(testoutput.Source("cluster/foo/crb.yaml")),
			),
			fake.ClusterRoleBindingAtPath("cluster/foo/crb.yaml")),
		parsertest.Success("A subdir of clusterregistry/ is ok",
			testoutput.NewAllConfigs(t),
			fake.ClusterAtPath("clusterregistry/foo/cluster.yaml")))

	test.RunAll(t)
}

func TestParserVet(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Success("A subdir of system/ is ok",
			testoutput.NewAllConfigs(t),
			fake.HierarchyConfigAtPath("system/sub/hc.yaml")),
		parsertest.Failure("Objects in non-namespaces/ with an invalid label is an error",
			metadata.IllegalLabelDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				core.Label("configmanagement.gke.io/illegal-label", "true")),
		),
		parsertest.Failure("Objects in non-namespaces/ with an invalid annotation is an error",
			metadata.IllegalAnnotationDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				core.Annotation("configmanagement.gke.io/illegal-annotation", "true")),
		),
	)

	test.RunAll(t)
}
