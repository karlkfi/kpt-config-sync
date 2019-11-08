package filesystem_test

import (
	"encoding/json"
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/parsertest"
	"github.com/google/nomos/testing/testoutput"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
)

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

func scopedResourceQuota(opts ...core.MetaMutator) *corev1.ResourceQuota {
	obj := fake.ResourceQuotaObject(opts...)
	obj.Spec.Scopes = []corev1.ResourceQuotaScope{"Terminating"}
	obj.Spec.ScopeSelector = &corev1.ScopeSelector{
		MatchExpressions: []corev1.ScopedResourceSelectorRequirement{
			{Operator: "In", ScopeName: "PriorityClass"},
		},
	}
	return obj
}

func TestParseRepo(t *testing.T) {
	test := parsertest.Test{
		NewParser: parsertest.NewParser,
		TestCases: []parsertest.TestCase{
			parsertest.Failure("missing repo",
				vet.MissingRepoErrorCode),
			parsertest.Failure("invalid repo version",
				vet.UnsupportedRepoSpecVersionCode,
				fake.Repo(fake.RepoVersion("0.0.0"))),
		},
	}

	test.RunAll(t)
}

func TestParserVetErrors(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Success("ResourceQuota with scope and normal quota",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/bar", nil, scopedResourceQuota(testoutput.Source("namespaces/bar/rq.yaml"))),
				),
				Syncs: testoutput.Syncs(kinds.ResourceQuota()),
			},
			fake.Namespace("namespaces/bar"),
			fake.FileObject(scopedResourceQuota(), "namespaces/bar/rq.yaml"),
		),
		parsertest.Failures("ResourceQuota with scope and hierarchical quota",
			[]string{vet.IllegalResourceQuotaFieldErrorCode, vet.IllegalResourceQuotaFieldErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
			fake.Namespace("namespaces/bar"),
			fake.FileObject(scopedResourceQuota(), "namespaces/bar/rq.yaml"),
		),
		parsertest.Success("Engineer CustomResourceDefinition",
			&namespaceconfig.AllConfigs{
				CRDClusterConfig: testoutput.CRDClusterConfig(engineerCRD(testoutput.Source("cluster/engineer-crd.yaml"))),
				Syncs:            testoutput.Syncs(kinds.CustomResourceDefinition()),
			},
			fake.FileObject(engineerCRD(), "cluster/engineer-crd.yaml"),
		),
		parsertest.Success("Valid to have Abstract Namespace with both Namespace and Abstract Namespace children",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/bar/foo", nil),
					testoutput.NamespaceConfig("", "namespaces/bar/qux/lym", nil),
				),
			},
			fake.Namespace("namespaces/bar/foo"),
			fake.Namespace("namespaces/bar/qux/lym")),
		parsertest.Failures("Namespace dir with Abstract Namespace child",
			[]string{
				vet.IllegalNamespaceSubdirectoryErrorCode,
				vet.UnsyncableResourcesErrorCode,
			},
			fake.Namespace("namespaces/bar"),
			fake.RoleAtPath("namespaces/bar/foo/rb.yaml"),
		),
		parsertest.Failure("Unsyncable resources because never instantiated",
			vet.UnsyncableResourcesErrorCode,
			fake.RoleBindingAtPath("namespaces/rb.yaml"),
		),
		parsertest.Failure("Abstract Namespace dir with uninheritable Rolebinding",
			vet.IllegalAbstractNamespaceObjectKindErrorCode,
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeNone)),
			fake.RoleBindingAtPath("namespaces/rb.yaml"),
			fake.Namespace("namespaces/bar"),
		),
		parsertest.Success("NamespaceSelector",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/bar/prod-ns", core.Label("env", "prod"),
						fake.RoleBindingObject(core.Name("sre"),
							inlinedNamespaceSelectorAnnotation(t, namespaceSelector("sre-supported", "env", "prod")),
							testoutput.Source("namespaces/bar/rb.yaml"),
						),
					),
					testoutput.NamespaceConfig("", "namespaces/bar/test-ns", core.Label("env", "test")),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			fake.FileObject(namespaceSelector("sre-supported", "env", "prod"), "namespaces/bar/ns-selector.yaml"),
			fake.RoleBindingAtPath("namespaces/bar/rb.yaml", core.Name("sre"), namespaceSelectorAnnotation("sre-supported")),
			fake.Namespace("namespaces/bar/prod-ns", core.Label("env", "prod")),
			fake.Namespace("namespaces/bar/test-ns", core.Label("env", "test")),
		),
		parsertest.Success("minimal repo",
			nil),
		parsertest.Success("Multiple resources with HierarchyConfigs",
			nil,
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeInherit)),
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.Role(), v1.HierarchyModeInherit)),
		),
		parsertest.Failure("Namespaces dir with Namespace",
			vet.IllegalTopLevelNamespaceErrorCode,
			fake.Namespace("namespaces")),
		parsertest.Success("Namespaces dir with ResourceQuota and hierarchical quota inheritance",
			&namespaceconfig.AllConfigs{
				ClusterConfig: testoutput.ClusterConfig(
					fake.HierarchicalQuotaObject(fake.HierarchicalQuotaRoot(
						fake.HierarchicalQuotaNode("namespaces", v1.HierarchyNodeAbstractNamespace,
							resourceQuotaObject(core.Name(resourcequota.ResourceQuotaObjectName), testoutput.Source("namespaces/rq.yaml")),
							fake.HierarchicalQuotaNode("bar", v1.HierarchyNodeNamespace,
								resourceQuotaObject(core.Name(resourcequota.ResourceQuotaObjectName), testoutput.Source("namespaces/rq.yaml"), core.Labels(resourcequota.NewConfigManagementQuotaLabels())))),
					)),
				),
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/bar", nil,
						resourceQuotaObject(testoutput.Source("namespaces/rq.yaml"), core.Name(resourcequota.ResourceQuotaObjectName), core.Labels(resourcequota.NewConfigManagementQuotaLabels())))),
				Syncs: testoutput.Syncs(kinds.ResourceQuota(), kinds.HierarchicalQuota()),
			},
			fake.HierarchyConfigAtPath("system/rq.yaml", fake.HierarchyConfigMeta(core.Name("resourcequota")), fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
			fake.FileObject(resourceQuotaObject(core.Name("pod-quota")), "namespaces/rq.yaml"),
			fake.Namespace("namespaces/bar")),
		parsertest.Success("Namespace with multiple inherited RoleBindings",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/foo", nil,
						fake.RoleBindingObject(core.Name("alice"), testoutput.Source("namespaces/rb-1.yaml")),
						fake.RoleBindingObject(core.Name("bob"), testoutput.Source("namespaces/rb-2.yaml"))),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			fake.RoleBindingAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.RoleBindingAtPath("namespaces/rb-2.yaml", core.Name("bob")),
			fake.Namespace("namespaces/foo")),
		parsertest.Failure("Cluster-scoped objects must not collide",
			vet.MetadataNameCollisionErrorCode,
			fake.ClusterRoleAtPath("cluster/cr-1.yaml", core.Name("alice")),
			fake.ClusterRoleAtPath("cluster/cr-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Namespaces must be uniquely named",
			vet.DuplicateDirectoryNameErrorCode,
			fake.Namespace("namespaces/foo/bar"),
			fake.Namespace("namespaces/qux/bar"),
		),
		parsertest.Success("Two abstract Namespace dirs with non-unique names are allowed.",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/foo/foo/bar", nil),
				),
			},
			fake.Namespace("namespaces/foo/foo/bar"),
		),
		parsertest.Success("An abstract namespace and a leaf namespace may share a name",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/bar/foo", nil),
					testoutput.NamespaceConfig("", "namespaces/foo/bar", nil),
				),
			},
			fake.Namespace("namespaces/bar/foo"),
			fake.Namespace("namespaces/foo/bar"),
		),
		parsertest.Success("kube-* is a system dir but is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/kube-something", nil)),
			},
			fake.Namespace("namespaces/kube-something"),
		),
		parsertest.Success("kube-system is a system dir but is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/kube-system", nil)),
			},
			fake.Namespace("namespaces/kube-system"),
		),
		parsertest.Success("Default namespace is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(testoutput.NamespaceConfig("", "namespaces/default", nil)),
			},
			fake.Namespace("namespaces/default"),
		),
		parsertest.Failure("Dir name invalid",
			vet.InvalidDirectoryNameErrorCode,
			fake.Namespace("namespaces/foo bar"),
		),
		parsertest.Failure("Namespace with NamespaceSelector annotation is invalid",
			vet.IllegalNamespaceAnnotationErrorCode,
			fake.Namespace("namespaces/bar", core.Annotation(v1.NamespaceSelectorAnnotationKey, "prod")),
		),
		parsertest.Failure("NamespaceSelector may not have clusterSelector annotations",
			vet.NamespaceSelectorMayNotHaveAnnotationCode,
			fake.FileObject(clusterSelectorObject("prod-cluster", "env", "prod"), "clusterregistry/cs.yaml"),
			fake.NamespaceSelector(clusterSelectorAnnotation("prod-cluster")),
		),
		parsertest.Failure("Namespace-scoped object in cluster/ dir",
			vet.IllegalKindInClusterErrorCode,
			fake.RoleBindingAtPath("cluster/rb.yaml"),
		),
		parsertest.Failure("Illegal annotation definition is an error",
			vet.IllegalAnnotationDefinitionErrorCode,
			fake.ClusterRole(core.Annotation("configmanagement.gke.io/stuff", "prod")),
		),
		parsertest.Failure("Illegal label definition is an error",
			vet.IllegalLabelDefinitionErrorCode,
			fake.ClusterRole(core.Label("configmanagement.gke.io/stuff", "prod")),
		),
		parsertest.Failure("Illegal object declaration in system/ is an error",
			vet.IllegalKindInSystemErrorCode,
			fake.RoleAtPath("system/role.yaml"),
		),
		parsertest.Failure("Duplicate Repo definitions is an error",
			vet.MultipleSingletonsErrorCode,
			fake.Repo(),
			fake.Repo(),
		),
		parsertest.Failure("Name collision in namespace",
			vet.MetadataNameCollisionErrorCode,
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/foo/role-1.yaml", core.Name("alice")),
			fake.RoleAtPath("namespaces/foo/role-2.yaml", core.Name("alice")),
		),
		parsertest.Success("No name collision if different types",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/foo", nil,
						fake.RoleObject(core.Name("alice"), testoutput.Source("namespaces/foo/role.yaml")),
						fake.RoleBindingObject(core.Name("alice"), testoutput.Source("namespaces/foo/rolebinding.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.Role(), kinds.RoleBinding()),
			},
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/foo/role.yaml", core.Name("alice")),
			fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml", core.Name("alice")),
		),
		parsertest.Failure("Name collision in child node",
			vet.MetadataNameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleAtPath("namespaces/foo/bar/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Name collision in grandchild node",
			vet.MetadataNameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/bar/qux"),
			fake.RoleAtPath("namespaces/foo/bar/qux/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Success("No name collision in sibling nodes",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig("", "namespaces/foo/bar", nil,
						fake.RoleBindingObject(core.Name("alice"), testoutput.Source("namespaces/foo/bar/rb-1.yaml"))),
					testoutput.NamespaceConfig("", "namespaces/foo/qux", nil,
						fake.RoleBindingObject(core.Name("alice"), testoutput.Source("namespaces/foo/qux/rb-2.yaml"))),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleBindingAtPath("namespaces/foo/bar/rb-1.yaml", core.Name("alice")),
			fake.Namespace("namespaces/foo/qux"),
			fake.RoleBindingAtPath("namespaces/foo/qux/rb-2.yaml", core.Name("alice")),
		),
		parsertest.Failure("Empty string name is an error",
			vet.MissingObjectNameErrorCode,
			fake.ClusterRole(core.Name(""))),
		parsertest.Failure("Specifying resourceVersion is an error",
			syntax.IllegalFieldsInConfigErrorCode,
			fake.ClusterRole(resourceVersion("999")),
		),
		parsertest.Failures("Repo outside system/ is an error",
			[]string{
				vet.IllegalSystemResourcePlacementErrorCode,
				vet.UnsupportedObjectErrorCode,
			},
			fake.RepoAtPath("cluster/repo.yaml")),
		parsertest.Failures("HierarchyConfig outside system/ is an error",
			[]string{
				vet.IllegalSystemResourcePlacementErrorCode,
				vet.UnsupportedObjectErrorCode,
			},
			fake.HierarchyConfigAtPath("cluster/hc.yaml")),
		parsertest.Failures("HierarchyConfig contains a CRD",
			[]string{
				vet.UnsupportedResourceInHierarchyConfigErrorCode,
				vet.ClusterScopedResourceInHierarchyConfigErrorCode,
			},
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.NamespaceConfig(), v1.HierarchyModeInherit)),
		),
		parsertest.Failures("HierarchyConfig contains a Namespace",
			[]string{
				vet.UnsupportedResourceInHierarchyConfigErrorCode,
				vet.ClusterScopedResourceInHierarchyConfigErrorCode,
			},
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.NamespaceConfig(), v1.HierarchyModeInherit)),
		),
		parsertest.Failures("HierarchyConfig contains a NamespaceConfig",
			[]string{
				vet.UnsupportedResourceInHierarchyConfigErrorCode,
				vet.ClusterScopedResourceInHierarchyConfigErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.NamespaceConfig(), v1.HierarchyModeInherit)),
		),
		parsertest.Failures("HierarchyConfig contains a Sync",
			[]string{
				vet.UnsupportedResourceInHierarchyConfigErrorCode,
				vet.ClusterScopedResourceInHierarchyConfigErrorCode},
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.Sync(), v1.HierarchyModeInherit)),
		),
		parsertest.Failure("Invalid name for HierarchyConfig",
			metadata.InvalidMetadataNameErrorCode,
			fake.HierarchyConfig(fake.HierarchyConfigMeta(core.Name("RBAC"))),
		),
		parsertest.Failure("Illegal Namespace in clusterregistry/",
			vet.IllegalKindInClusterregistryErrorCode,
			fake.Namespace("clusterregistry"),
		),
		parsertest.Failure("Illegal NamespaceSelector in Namespace directory.",
			vet.IllegalKindInNamespacesErrorCode,
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
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(testoutput.InCluster(prodCluster), testoutput.Source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Resource without selector always exists 2",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(testoutput.InCluster(devCluster), testoutput.Source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Namespace resource selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(prodSelectorAnnotationInlined, testoutput.InCluster(prodCluster), testoutput.Source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace resource not selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(devCluster, "namespaces/bar", nil),
				),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Namespace selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(prodCluster, "namespaces/bar", prodSelectorAnnotationInlined,
						fake.RoleBindingObject(testoutput.InCluster(prodCluster), testoutput.Source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace not selected",
			nil,
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar", prodSelectorAnnotation),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Cluster resource selected",
			&namespaceconfig.AllConfigs{
				ClusterConfig: testoutput.ClusterConfig(
					fake.ClusterRoleBindingObject(prodSelectorAnnotationInlined, testoutput.InCluster(prodCluster), testoutput.Source("cluster/crb.yaml")),
				),
				Syncs: testoutput.Syncs(kinds.ClusterRoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Cluster resource not selected",
			nil,
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.ClusterRoleBinding(prodSelectorAnnotation),
		).ForCluster(devCluster),
		parsertest.Success("Abstract Namespace resouce selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(prodCluster, "namespaces/foo/bar", nil,
						fake.ConfigMapObject(prodSelectorAnnotationInlined, testoutput.InCluster(prodCluster), testoutput.Source("namespaces/foo/configmap.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.ConfigMap()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.HierarchyConfig(fake.HierarchyConfigResource(kinds.ConfigMap(), v1.HierarchyModeInherit)),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/foo/bar"),
			fake.ConfigMapAtPath("namespaces/foo/configmap.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Colliding resources selected to different clusters may coexist",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: testoutput.NamespaceConfigs(
					testoutput.NamespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(devSelectorAnnotationInlined, testoutput.InCluster(devCluster), testoutput.Source("namespaces/bar/rolebinding-2.yaml")),
					),
				),
				Syncs: testoutput.Syncs(kinds.RoleBinding()),
			},
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
			vet.ObjectHasUnknownClusterSelectorCode,
			fake.Namespace("namespaces/foo", clusterSelectorAnnotation("does-not-exist")),
		),
		parsertest.Failure(
			"A cluster object that has a cluster selector annotation for nonexistent cluster is an error",
			vet.ObjectHasUnknownClusterSelectorCode,
			fake.ClusterRole(clusterSelectorAnnotation("does-not-exist")),
		),
		parsertest.Success("A subdir of cluster/ is ok",
			&namespaceconfig.AllConfigs{
				ClusterConfig: testoutput.ClusterConfig(
					fake.ClusterRoleBindingObject(testoutput.Source("cluster/foo/crb.yaml")),
				),
				Syncs: testoutput.Syncs(kinds.ClusterRoleBinding()),
			},
			fake.ClusterRoleBindingAtPath("cluster/foo/crb.yaml")),
		parsertest.Success("A subdir of clusterregistry/ is ok",
			&namespaceconfig.AllConfigs{},
			fake.ClusterAtPath("clusterregistry/foo/cluster.yaml")))

	test.RunAll(t)
}

func TestParserVet(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Success("A subdir of system/ is ok",
			&namespaceconfig.AllConfigs{},
			fake.HierarchyConfigAtPath("system/sub/hc.yaml")),
		parsertest.Failure("Objects in non-namespaces/ with an invalid label is an error",
			vet.IllegalLabelDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(core.Label("configmanagement.gke.io/illegal-label", "true"))),
		),
		parsertest.Failure("Objects in non-namespaces/ with an invalid annotation is an error",
			vet.IllegalAnnotationDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(core.Annotation("configmanagement.gke.io/illegal-annotation", "true"))),
		),
	)

	test.RunAll(t)
}
