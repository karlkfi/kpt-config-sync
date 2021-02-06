package filesystem_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ft "github.com/google/nomos/pkg/importer/filesystem/filesystemtest"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/parsertest"
	"github.com/google/nomos/testing/testoutput"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	importToken = visitortesting.ImportToken
	loadTime    = metav1.Time{}
)

func flatClusterSelector(name, key, value string) ast.FileObject {
	cs := fake.ClusterSelectorObject(core.Name(name))
	cs.Spec.Selector.MatchLabels = map[string]string{key: value}
	return fake.FileObject(cs, name+"_cs.yaml")
}

func inlineClusterSelector(clusterName string) core.MetaMutator {
	return core.Annotation(v1alpha1.ClusterNameSelectorAnnotationKey, clusterName)
}

func flatNamespaceSelector(name, key, value string) ast.FileObject {
	cs := fake.NamespaceSelectorObject(core.Name(name))
	cs.Spec.Selector.MatchLabels = map[string]string{key: value}
	return fake.FileObject(cs, name+"_cs.yaml")
}

func TestRawParser_Parse(t *testing.T) {
	testCases := []struct {
		testName         string
		clusterName      string
		defaultNamespace string
		objects          []ast.FileObject
		syncedCRDs       []*v1beta1.CustomResourceDefinition
		expected         *namespaceconfig.AllConfigs
	}{
		{
			testName: "empty returns empty",
			expected: testoutput.NewAllConfigs(),
		},
		{
			testName: "cluster-scoped object",
			objects: []ast.FileObject{
				fake.ClusterRole(),
			},
			expected: testoutput.NewAllConfigs(fake.ClusterRole()),
		},
		{
			testName: "preserves Namespace",
			objects: []ast.FileObject{
				fake.Role(core.Namespace("foo")),
				fake.RoleBinding(core.Namespace("bar")),
			},
			expected: testoutput.NewAllConfigs(
				fake.Role(core.Namespace("foo")),
				fake.RoleBinding(core.Namespace("bar")),
			),
		},
		{
			testName: "allows synced CRDs",
			objects: []ast.FileObject{
				fake.FileObject(fakeCRD(kinds.Anvil()), "crd.yaml"),
				fake.AnvilAtPath("anvil.yaml"),
			},
			expected: testoutput.NewAllConfigs(
				fake.FileObject(fakeCRD(kinds.Anvil()), "crd.yaml"),
				fake.AnvilAtPath("anvil.yaml"),
			),
		},
		{
			testName:    "resolves ClusterSelectors",
			clusterName: "prod",
			objects: []ast.FileObject{
				fake.Cluster(core.Name("prod"), core.Label("environment", "prod")),
				flatClusterSelector("prod-only", "environment", "prod"),
				flatClusterSelector("dev-only", "environment", "dev"),
				fake.ClusterRole(core.Name("prod-owner"), inlineClusterSelector("prod")),
				fake.ClusterRole(core.Name("prod-admin"), core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "prod-only")),
				fake.ClusterRole(core.Name("dev-admin"), core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "dev-only")),
				fake.ClusterRole(core.Name("dev-owner"), inlineClusterSelector("dev")),
				fake.Namespace("prod-shipping", inlineClusterSelector("prod")),
				fake.Role(core.Name("prod-sre"), core.Namespace("prod-shipping")),
				fake.Namespace("dev-shipping", core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "dev-only")),
				fake.Role(core.Name("dev-sre"), core.Namespace("dev-shipping")),
			},
			expected: testoutput.NewAllConfigs(
				fake.ClusterRole(core.Name("prod-owner"), core.Annotation(v1.ClusterNameAnnotationKey, "prod"), inlineClusterSelector("prod")),
				fake.ClusterRole(core.Name("prod-admin"), core.Annotation(v1.ClusterNameAnnotationKey, "prod"), core.Annotation(v1.LegacyClusterSelectorAnnotationKey, "prod-only")),
				fake.Namespace("prod-shipping", core.Annotation(v1.ClusterNameAnnotationKey, "prod"), inlineClusterSelector("prod")),
				fake.Role(core.Name("prod-sre"), core.Namespace("prod-shipping"), core.Annotation(v1.ClusterNameAnnotationKey, "prod")),
			),
		},
		{
			testName:    "resolves NamespaceSelectors",
			clusterName: "prod",
			objects: []ast.FileObject{
				flatNamespaceSelector("sre", "sre-supported", "true"),
				fake.Namespace("prod-shipping", core.Label("sre-supported", "true")),
				fake.Namespace("dev-shipping"),
				fake.Role(core.Name("sre-role"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "sre")),
			},
			expected: testoutput.NewAllConfigs(
				fake.Namespace("prod-shipping", core.Label("sre-supported", "true"), core.Annotation(v1.ClusterNameAnnotationKey, "prod")),
				fake.Namespace("dev-shipping", core.Annotation(v1.ClusterNameAnnotationKey, "prod")),
				fake.Role(core.Name("sre-role"), core.Namespace("prod-shipping"), core.Annotation(v1.ClusterNameAnnotationKey, "prod"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "sre")),
			),
		},
		{
			testName: "sets Namespace default if scope unset",
			objects: []ast.FileObject{
				fake.Role(core.Namespace("")),
			},
			expected: testoutput.NewAllConfigs(
				fake.Role(core.Namespace(metav1.NamespaceDefault)),
			),
		},
		{
			testName:         "sets scope if scope set",
			defaultNamespace: "shipping",
			objects: []ast.FileObject{
				fake.Role(core.Namespace("")),
			},
			expected: testoutput.NewAllConfigs(
				fake.Role(core.Namespace("shipping")),
			),
		},
		{
			testName: "don't fail on Anthos Validator type without CRD",
			objects: []ast.FileObject{
				fake.Namespace("namespaces/foo"),
				fake.Unstructured(schema.GroupVersionKind{Group: "anthos.cloud.google.com", Version: "v1alpha1", Kind: "Validator"}, core.Namespace("foo"),
					syncertest.ManagementDisabled),
			},
			expected: testoutput.NewAllConfigs(
				fake.Namespace("namespaces/foo"),
				fake.Unstructured(schema.GroupVersionKind{Group: "anthos.cloud.google.com", Version: "v1alpha1", Kind: "Validator"}, core.Namespace("foo"),
					syncertest.ManagementDisabled),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			f := ft.NewTestDiscoveryClient(parsertest.CRDsToAPIGroupResources(tc.syncedCRDs))

			root, err := cmpath.AbsoluteSlash("/")
			if err != nil {
				t.Fatal(err)
			}
			r := ft.NewFakeReader(root, tc.objects)

			policyDir := cmpath.RelativeSlash("/")
			fps := reader.FilePaths{RootDir: root, PolicyDir: policyDir, Files: r.ToFileList()}

			scope := tc.defaultNamespace
			if scope == "" {
				scope = metav1.NamespaceDefault
			}

			p := filesystem.NewRawParser(r, true, scope, declared.RootReconciler)
			builder := discovery.ScoperBuilder(f)
			coreObjects, err := p.Parse(tc.clusterName, nil, builder, fps)
			fileObjects := filesystem.AsFileObjects(coreObjects)
			result := namespaceconfig.NewAllConfigs(importToken, loadTime, fileObjects)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestRawParser_ParseErrors(t *testing.T) {
	testCases := []struct {
		name              string
		objects           []ast.FileObject
		syncedCRDs        []*v1beta1.CustomResourceDefinition
		expectedErrorCode string
	}{
		{
			name: "duplicate objects",
			objects: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
			},
			expectedErrorCode: nonhierarchical.NameCollisionErrorCode,
		},
		{
			name: "error on illegal CRD removal",
			objects: []ast.FileObject{
				fake.AnvilAtPath("anvil.yaml"),
			},
			syncedCRDs:        []*v1beta1.CustomResourceDefinition{fakeCRD(kinds.Anvil())},
			expectedErrorCode: nonhierarchical.UnsupportedCRDRemovalErrorCode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := ft.NewTestDiscoveryClient(parsertest.CRDsToAPIGroupResources(tc.syncedCRDs))

			root, err := cmpath.AbsoluteSlash("/")
			if err != nil {
				t.Fatal(err)
			}
			r := ft.NewFakeReader(root, tc.objects)
			p := filesystem.NewRawParser(r, true, metav1.NamespaceDefault, declared.RootReconciler)

			policyDir := cmpath.RelativeSlash("/")
			fps := reader.FilePaths{RootDir: root, PolicyDir: policyDir, Files: r.ToFileList()}
			builder := discovery.ScoperBuilder(f)
			_, err2 := p.Parse("", tc.syncedCRDs, builder, fps)
			if err2 == nil {
				t.Fatal("expected error")
			}

			errs := err2.Errors()
			if len(errs) != 1 {
				t.Fatalf("expected only one error, got %+v", errs)
			}

			code := errs[0].Code()
			if code != tc.expectedErrorCode {
				t.Fatalf("expected code %q, got %q", tc.expectedErrorCode, code)
			}
		})
	}
}
