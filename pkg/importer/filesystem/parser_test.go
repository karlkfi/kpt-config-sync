package filesystem_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/backend"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/parsertest"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func engineerCRD(opts ...object.MetaMutator) *v1beta1.CustomResourceDefinition {
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

type parserTestCase struct {
	testName                 string
	testFiles                fstesting.FileContentMap
	expectedNamespaceConfigs map[string]v1.NamespaceConfig
	expectedSyncs            map[string]v1.Sync
	expectedErrorCodes       []string
}

func scopedResourceQuota(opts ...object.MetaMutator) *corev1.ResourceQuota {
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
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/bar", nil, scopedResourceQuota(source("namespaces/bar/rq.yaml"))),
				),
				Syncs: syncs(kinds.ResourceQuota()),
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
				CRDClusterConfig: crdClusterConfig(engineerCRD(source("cluster/engineer-crd.yaml"))),
				Syncs:            syncs(kinds.CustomResourceDefinition()),
			},
			fake.FileObject(engineerCRD(), "cluster/engineer-crd.yaml"),
		),
		parsertest.Success("Valid to have Abstract Namespace with both Namespace and Abstract Namespace children",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/bar/foo", nil),
					namespaceConfig("", "namespaces/bar/qux/lym", nil),
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
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/bar/prod-ns", object.Label("env", "prod"),
						fake.RoleBindingObject(object.Name("sre"),
							inlinedNamespaceSelectorAnnotation(t, namespaceSelector("sre-supported", "env", "prod")),
							source("namespaces/bar/rb.yaml"),
						),
					),
					namespaceConfig("", "namespaces/bar/test-ns", object.Label("env", "test")),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			fake.FileObject(namespaceSelector("sre-supported", "env", "prod"), "namespaces/bar/ns-selector.yaml"),
			fake.RoleBindingAtPath("namespaces/bar/rb.yaml", object.Name("sre"), namespaceSelectorAnnotation("sre-supported")),
			fake.Namespace("namespaces/bar/prod-ns", object.Label("env", "prod")),
			fake.Namespace("namespaces/bar/test-ns", object.Label("env", "test")),
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
				ClusterConfig: clusterConfig(
					fake.HierarchicalQuotaObject(fake.HierarchicalQuotaRoot(
						fake.HierarchicalQuotaNode("namespaces", v1.HierarchyNodeAbstractNamespace,
							resourceQuotaObject(object.Name(resourcequota.ResourceQuotaObjectName), source("namespaces/rq.yaml")),
							fake.HierarchicalQuotaNode("bar", v1.HierarchyNodeNamespace,
								resourceQuotaObject(object.Name(resourcequota.ResourceQuotaObjectName), source("namespaces/rq.yaml"), object.Labels(resourcequota.NewConfigManagementQuotaLabels())))),
					)),
				),
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/bar", nil,
						resourceQuotaObject(source("namespaces/rq.yaml"), object.Name(resourcequota.ResourceQuotaObjectName), object.Labels(resourcequota.NewConfigManagementQuotaLabels())))),
				Syncs: syncs(kinds.ResourceQuota(), kinds.HierarchicalQuota()),
			},
			fake.HierarchyConfigAtPath("system/rq.yaml", fake.HierarchyConfigMeta(object.Name("resourcequota")), fake.HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
			fake.FileObject(resourceQuotaObject(object.Name("pod-quota")), "namespaces/rq.yaml"),
			fake.Namespace("namespaces/bar")),
		parsertest.Success("Namespace with multiple inherited RoleBindings",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/foo", nil,
						fake.RoleBindingObject(object.Name("alice"), source("namespaces/rb-1.yaml")),
						fake.RoleBindingObject(object.Name("bob"), source("namespaces/rb-2.yaml"))),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			fake.RoleBindingAtPath("namespaces/rb-1.yaml", object.Name("alice")),
			fake.RoleBindingAtPath("namespaces/rb-2.yaml", object.Name("bob")),
			fake.Namespace("namespaces/foo")),
		parsertest.Failure("Cluster-scoped objects must not collide",
			vet.MetadataNameCollisionErrorCode,
			fake.ClusterRoleAtPath("cluster/cr-1.yaml", object.Name("alice")),
			fake.ClusterRoleAtPath("cluster/cr-2.yaml", object.Name("alice")),
		),
		parsertest.Failure("Namespaces must be uniquely named",
			vet.DuplicateDirectoryNameErrorCode,
			fake.Namespace("namespaces/foo/bar"),
			fake.Namespace("namespaces/qux/bar"),
		),
		parsertest.Success("Two abstract Namespace dirs with non-unique names are allowed.",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/foo/foo/bar", nil),
				),
			},
			fake.Namespace("namespaces/foo/foo/bar"),
		),
		parsertest.Success("An abstract namespace and a leaf namespace may share a name",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/bar/foo", nil),
					namespaceConfig("", "namespaces/foo/bar", nil),
				),
			},
			fake.Namespace("namespaces/bar/foo"),
			fake.Namespace("namespaces/foo/bar"),
		),
		parsertest.Success("kube-* is a system dir but is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(namespaceConfig("", "namespaces/kube-something", nil)),
			},
			fake.Namespace("namespaces/kube-something"),
		),
		parsertest.Success("kube-system is a system dir but is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(namespaceConfig("", "namespaces/kube-system", nil)),
			},
			fake.Namespace("namespaces/kube-system"),
		),
		parsertest.Success("Default namespace is allowed",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(namespaceConfig("", "namespaces/default", nil)),
			},
			fake.Namespace("namespaces/default"),
		),
		parsertest.Failure("Dir name invalid",
			vet.InvalidDirectoryNameErrorCode,
			fake.Namespace("namespaces/foo bar"),
		),
		parsertest.Failure("Namespace with NamespaceSelector annotation is invalid",
			vet.IllegalNamespaceAnnotationErrorCode,
			fake.Namespace("namespaces/bar", object.Annotation(v1.NamespaceSelectorAnnotationKey, "prod")),
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
			fake.ClusterRole(object.Annotation("configmanagement.gke.io/stuff", "prod")),
		),
		parsertest.Failure("Illegal label definition is an error",
			vet.IllegalLabelDefinitionErrorCode,
			fake.ClusterRole(object.Label("configmanagement.gke.io/stuff", "prod")),
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
			fake.RoleAtPath("namespaces/foo/role-1.yaml", object.Name("alice")),
			fake.RoleAtPath("namespaces/foo/role-2.yaml", object.Name("alice")),
		),
		parsertest.Success("No name collision if different types",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/foo", nil,
						fake.RoleObject(object.Name("alice"), source("namespaces/foo/role.yaml")),
						fake.RoleBindingObject(object.Name("alice"), source("namespaces/foo/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.Role(), kinds.RoleBinding()),
			},
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/foo/role.yaml", object.Name("alice")),
			fake.RoleBindingAtPath("namespaces/foo/rolebinding.yaml", object.Name("alice")),
		),
		parsertest.Failure("Name collision in child node",
			vet.MetadataNameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", object.Name("alice")),
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleAtPath("namespaces/foo/bar/rb-2.yaml", object.Name("alice")),
		),
		parsertest.Failure("Name collision in grandchild node",
			vet.MetadataNameCollisionErrorCode,
			fake.RoleAtPath("namespaces/rb-1.yaml", object.Name("alice")),
			fake.Namespace("namespaces/foo/bar/qux"),
			fake.RoleAtPath("namespaces/foo/bar/qux/rb-2.yaml", object.Name("alice")),
		),
		parsertest.Success("No name collision in sibling nodes",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig("", "namespaces/foo/bar", nil,
						fake.RoleBindingObject(object.Name("alice"), source("namespaces/foo/bar/rb-1.yaml"))),
					namespaceConfig("", "namespaces/foo/qux", nil,
						fake.RoleBindingObject(object.Name("alice"), source("namespaces/foo/qux/rb-2.yaml"))),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			fake.Namespace("namespaces/foo/bar"),
			fake.RoleBindingAtPath("namespaces/foo/bar/rb-1.yaml", object.Name("alice")),
			fake.Namespace("namespaces/foo/qux"),
			fake.RoleBindingAtPath("namespaces/foo/qux/rb-2.yaml", object.Name("alice")),
		),
		parsertest.Failure("Empty string name is an error",
			vet.MissingObjectNameErrorCode,
			fake.ClusterRole(object.Name(""))),
		parsertest.Failure("Specifying resourceVersion is an error",
			vet.IllegalFieldsInConfigErrorCode,
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
			vet.InvalidMetadataNameErrorCode,
			fake.HierarchyConfig(fake.HierarchyConfigMeta(object.Name("RBAC"))),
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
		parsertest.Failure("Resource with OwnerReferences specified",
			vet.IllegalFieldsInConfigErrorCode,
			fake.Namespace("namespaces/foo", object.OwnerReference("owner", "1", kinds.Cluster())),
		),
	)

	test.RunAll(t)
}

func (tc *parserTestCase) Run(t *testing.T) {
	d := newTestDir(t)
	defer d.remove(t)

	if glog.V(6) {
		glog.Infof("Testcase: %+v", spew.Sdump(tc))
	}

	for k, v := range tc.testFiles {
		d.createTestFile(k, v, t)
	}

	f := fstesting.NewTestClientGetter(t)
	defer func() {
		if err := f.Cleanup(); err != nil {
			t.Fatal(errors.Wrap(err, "could not clean up"))
		}
	}()

	var err error
	rootPath, err := cmpath.NewRoot(cmpath.FromOS(d.rootDir))
	if err != nil {
		t.Error(err)
	}

	p := filesystem.NewParser(
		f,
		filesystem.ParserOpt{
			Vet:       true,
			Validate:  true,
			Extension: &filesystem.NomosVisitorProvider{},
			RootPath:  rootPath,
		},
	)
	actualConfigs, mErr := p.Parse("", &namespaceconfig.AllConfigs{}, time.Time{}, "")

	vettesting.ExpectErrors(tc.expectedErrorCodes, mErr, t)
	if mErr != nil || tc.expectedErrorCodes != nil {
		// We expected there to be an error, so no need to do config validation
		return
	}

	if tc.expectedNamespaceConfigs == nil {
		tc.expectedNamespaceConfigs = namespaceConfigs()
	}
	if tc.expectedSyncs == nil {
		tc.expectedSyncs = syncs()
	}

	expectedConfigs := &namespaceconfig.AllConfigs{
		NamespaceConfigs: tc.expectedNamespaceConfigs,
		ClusterConfig:    clusterConfig(),
		CRDClusterConfig: crdClusterConfig(),
		Syncs:            tc.expectedSyncs,
		Repo:             fake.RepoObject(),
	}
	if diff := cmp.Diff(expectedConfigs, actualConfigs, resourcequota.ResourceQuantityEqual(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Actual and expected configs didn't match: diff\n%v", diff)
	}
}

func resourceQuotaObject(opts ...object.MetaMutator) *corev1.ResourceQuota {
	obj := fake.ResourceQuotaObject(opts...)
	podQ, _ := resource.ParseQuantity("10")
	obj.Spec.Hard = map[corev1.ResourceName]resource.Quantity{corev1.ResourcePods: podQ}
	return obj
}

func namespaceSelector(name, key, value string, opts ...object.MetaMutator) *v1.NamespaceSelector {
	obj := fake.NamespaceSelectorObject(opts...)
	obj.Name = name
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func namespaceSelectorAnnotation(name string) object.MetaMutator {
	return object.Annotation(v1.NamespaceSelectorAnnotationKey, name)
}

func inlinedNamespaceSelectorAnnotation(t *testing.T, selector *v1.NamespaceSelector) object.MetaMutator {
	content, err := json.Marshal(selector)
	if err != nil {
		t.Error(err)
	}
	return object.Annotation(v1.NamespaceSelectorAnnotationKey, string(content))
}

func clusterSelectorAnnotation(value string) object.MetaMutator {
	return object.Annotation(v1.ClusterSelectorAnnotationKey, value)
}

func inlinedClusterSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) object.MetaMutator {
	content, err := json.Marshal(selector)
	if err != nil {
		t.Error(err)
	}
	return object.Annotation(v1.ClusterSelectorAnnotationKey, string(content))
}

func cluster(name string, opts ...object.MetaMutator) ast.FileObject {
	mutators := append(opts, object.Name(name))
	return fake.Cluster(mutators...)
}

func namespaceSelectorObject(name, key, value string) *v1.NamespaceSelector {
	obj := fake.NamespaceSelectorObject(object.Name(name))
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func clusterSelectorObject(name, key, value string) *v1.ClusterSelector {
	obj := fake.ClusterSelectorObject(object.Name(name))
	obj.Spec.Selector.MatchLabels = map[string]string{key: value}
	return obj
}

func inlinedSelectorAnnotation(t *testing.T, selector *v1.ClusterSelector) object.MetaMutator {
	return inlinedClusterSelectorAnnotation(t, selector)
}

func inCluster(clusterName string) object.MetaMutator {
	return object.Annotation(v1.ClusterNameAnnotationKey, clusterName)
}

func source(path string) object.MetaMutator {
	return object.Annotation(v1.SourcePathAnnotationKey, path)
}

func resourceVersion(version string) object.MetaMutator {
	return func(meta metav1.Object) {
		meta.SetResourceVersion(version)
	}
}

func crdClusterConfig(objects ...runtime.Object) *v1.ClusterConfig {
	config := fake.CRDClusterConfigObject()
	config.Spec.Resources = genericResources(objects...)
	return config
}

// clusterConfig generates a valid ClusterConfig to be put in AllConfigs given the set of hydrated
// cluster-scoped runtime.Objects.
func clusterConfig(objects ...runtime.Object) *v1.ClusterConfig {
	config := fake.ClusterConfigObject()
	config.Spec.Resources = genericResources(objects...)
	return config
}

// namespaceConfig generates a valid NamespaceConfig to be put in AllConfigs given the set of
// hydrated runtime.Objects for that Namespace.
func namespaceConfig(clusterName, dir string, opt object.MetaMutator, objects ...runtime.Object) v1.NamespaceConfig {
	config := fake.NamespaceConfigObject(fake.NamespaceConfigMeta(source(dir)))
	if clusterName != "" {
		inCluster(clusterName)(config)
	}
	config.Name = cmpath.FromSlash(dir).Base()
	config.Spec.Resources = genericResources(objects...)
	if opt != nil {
		opt(config)
	}
	return *config
}

// namespaceConfigs turns a list of NamespaceConfigs into the map AllConfigs requires.
func namespaceConfigs(ncs ...v1.NamespaceConfig) map[string]v1.NamespaceConfig {
	result := map[string]v1.NamespaceConfig{}
	for _, nc := range ncs {
		result[nc.Name] = nc
	}
	return result
}

// genericResources convers a list of runtime.Objects to the GenericResources array required for
// AllConfigs.
func genericResources(objects ...runtime.Object) []v1.GenericResources {
	var result []v1.GenericResources
	for _, obj := range objects {
		result = backend.AppendResource(result, obj)
	}
	return result
}

// syncs generates the sync map to be put in AllConfigs.
func syncs(gvks ...schema.GroupVersionKind) map[string]v1.Sync {
	result := map[string]v1.Sync{}
	for _, gvk := range gvks {
		result[groupKind(gvk)] = *fake.SyncObject(gvk.GroupKind())
	}
	return result
}

// groupKind factors out the two-line operation of getting the GroupKind string from a
// GroupVersionKind. The GroupKind.String() method has a pointer receiver, so
// gvk.GroupKind.String() is an error.
func groupKind(gvk schema.GroupVersionKind) string {
	gk := gvk.GroupKind()
	return strings.ToLower(gk.String())
}

// Test how the parser handles ClusterSelectors
func TestParseClusterSelector(t *testing.T) {
	prodCluster := "cluster-1"
	devCluster := "cluster-2"

	prodSelectorName := "sel-1"
	prodLabel := object.Label("environment", "prod")
	prodSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(prodSelectorName, "environment", "prod")
	}
	prodSelectorAnnotation := clusterSelectorAnnotation(prodSelectorName)
	prodSelectorAnnotationInlined := inlinedSelectorAnnotation(t, prodSelectorObject())

	devSelectorName := "sel-2"
	devLabel := object.Label("environment", "dev")
	devSelectorObject := func() *v1.ClusterSelector {
		return clusterSelectorObject(devSelectorName, "environment", "dev")
	}
	devSelectorAnnotation := clusterSelectorAnnotation(devSelectorName)
	devSelectorAnnotationInlined := inlinedSelectorAnnotation(t, devSelectorObject())

	test := parsertest.VetTest(
		parsertest.Success("Resource without selector always exists 1",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(prodCluster),
		parsertest.Success("Resource without selector always exists 2",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(inCluster(devCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml"),
		).ForCluster(devCluster),
		parsertest.Success("Namespace resource selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
			},
			cluster(prodCluster, prodLabel),
			cluster(devCluster, devLabel),
			fake.FileObject(prodSelectorObject(), "clusterregistry/cs.yaml"),

			fake.Namespace("namespaces/bar"),
			fake.RoleBindingAtPath("namespaces/bar/rolebinding.yaml", prodSelectorAnnotation),
		).ForCluster(prodCluster),
		parsertest.Success("Namespace resource not selected",
			&namespaceconfig.AllConfigs{
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil),
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
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/bar", prodSelectorAnnotationInlined,
						fake.RoleBindingObject(inCluster(prodCluster), source("namespaces/bar/rolebinding.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
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
				ClusterConfig: clusterConfig(
					fake.ClusterRoleBindingObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("cluster/crb.yaml")),
				),
				Syncs: syncs(kinds.ClusterRoleBinding()),
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
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(prodCluster, "namespaces/foo/bar", nil,
						fake.ConfigMapObject(prodSelectorAnnotationInlined, inCluster(prodCluster), source("namespaces/foo/configmap.yaml")),
					),
				),
				Syncs: syncs(kinds.ConfigMap()),
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
				NamespaceConfigs: namespaceConfigs(
					namespaceConfig(devCluster, "namespaces/bar", nil,
						fake.RoleBindingObject(devSelectorAnnotationInlined, inCluster(devCluster), source("namespaces/bar/rolebinding-2.yaml")),
					),
				),
				Syncs: syncs(kinds.RoleBinding()),
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
		))

	test.RunAll(t)
}

func TestParserVet(t *testing.T) {
	test := parsertest.VetTest(
		parsertest.Failure("A subdir of system is an error",
			vet.IllegalSubdirectoryErrorCode,
			fake.HierarchyConfigAtPath("system/sub/hc.yaml")),
		parsertest.Failure("Objects in non-namespaces/ with an invalid label is an error",
			vet.IllegalLabelDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(object.Label("configmanagement.gke.io/illegal-label", "true"))),
		),
		parsertest.Failure("Objects in non-namespaces/ with an invalid annotation is an error",
			vet.IllegalAnnotationDefinitionErrorCode,
			fake.HierarchyConfigAtPath("system/hc.yaml",
				fake.HierarchyConfigMeta(object.Annotation("configmanagement.gke.io/illegal-annotation", "true"))),
		),
	)

	test.RunAll(t)
}
