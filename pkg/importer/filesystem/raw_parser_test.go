package filesystem_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/parsertest"
	"github.com/google/nomos/testing/testoutput"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func flatNamespaceSelector(name, key, value string) ast.FileObject {
	cs := fake.NamespaceSelectorObject(core.Name(name))
	cs.Spec.Selector.MatchLabels = map[string]string{key: value}
	return fake.FileObject(cs, name+"_cs.yaml")
}

func TestRawParser_Parse(t *testing.T) {
	testCases := []struct {
		testName    string
		clusterName string
		objects     []ast.FileObject
		syncedCRDs  []*v1beta1.CustomResourceDefinition
		expected    *namespaceconfig.AllConfigs
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
				fake.ClusterRole(core.Name("prod-admin"), core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-only")),
				fake.ClusterRole(core.Name("dev-admin"), core.Annotation(v1.ClusterSelectorAnnotationKey, "dev-only")),
				fake.Namespace("prod-shipping", core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-only")),
				fake.Role(core.Name("prod-sre"), core.Namespace("prod-shipping")),
				fake.Namespace("dev-shipping", core.Annotation(v1.ClusterSelectorAnnotationKey, "dev-only")),
				fake.Role(core.Name("dev-sre"), core.Namespace("dev-shipping")),
			},
			expected: testoutput.NewAllConfigs(
				fake.ClusterRole(core.Name("prod-admin"), core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-only")),
				fake.Namespace("prod-shipping", core.Annotation(v1.ClusterSelectorAnnotationKey, "prod-only")),
				fake.Role(core.Name("prod-sre"), core.Namespace("prod-shipping")),
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
				fake.Namespace("prod-shipping", core.Label("sre-supported", "true")),
				fake.Namespace("dev-shipping"),
				fake.Role(core.Name("sre-role"), core.Namespace("prod-shipping"), core.Annotation(v1.NamespaceSelectorAnnotationKey, "sre")),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			f := fstesting.NewTestClientGetter(parsertest.CRDsToAPIGroupResources(tc.syncedCRDs))

			root, err := cmpath.AbsoluteSlash("/")
			if err != nil {
				t.Fatal(err)
			}
			r := fstesting.NewFakeReader(root, tc.objects)
			p := filesystem.NewRawParser(r, f)
			getSyncedCRDs := func() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
				return nil, nil
			}
			fileObjects, err := p.Parse(tc.clusterName, true, getSyncedCRDs,
				root, r.ToFileList())
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
			f := fstesting.NewTestClientGetter(parsertest.CRDsToAPIGroupResources(tc.syncedCRDs))

			root, err := cmpath.AbsoluteSlash("/")
			if err != nil {
				t.Fatal(err)
			}
			r := fstesting.NewFakeReader(root, tc.objects)
			p := filesystem.NewRawParser(r, f)

			getSyncedCRDs := func() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
				return tc.syncedCRDs, nil
			}
			_, err2 := p.Parse("", true, getSyncedCRDs,
				root, r.ToFileList())
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
