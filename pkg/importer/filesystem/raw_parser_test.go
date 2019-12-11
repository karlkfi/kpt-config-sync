package filesystem_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

type fakeReader []ast.FileObject

var _ filesystem.Reader = &fakeReader{}

func (r fakeReader) Read(_ cmpath.RootedPath) ([]ast.FileObject, status.MultiError) {
	return r, nil
}

func TestRawParser_Parse(t *testing.T) {
	testCases := []struct {
		name       string
		objects    []ast.FileObject
		syncedCRDs []*v1beta1.CustomResourceDefinition
		expected   *namespaceconfig.AllConfigs
	}{
		{
			name:     "empty returns empty",
			expected: testoutput.NewAllConfigs(),
		},
		{
			name: "cluster-scoped object",
			objects: []ast.FileObject{
				fake.ClusterRole(),
			},
			expected: testoutput.NewAllConfigs(fake.ClusterRole()),
		},
		{
			name: "preserves Namespace",
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
			name: "allows synced CRDs",
			objects: []ast.FileObject{
				fake.FileObject(fakeCRD(kinds.Anvil()), "crd.yaml"),
				fake.AnvilAtPath("anvil.yaml"),
			},
			expected: testoutput.NewAllConfigs(
				fake.FileObject(fakeCRD(kinds.Anvil()), "crd.yaml"),
				fake.AnvilAtPath("anvil.yaml"),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := fstesting.NewTestClientGetter(parsertest.CRDsToAPIGroupResources(tc.syncedCRDs))

			p := filesystem.NewRawParser(cmpath.Root{}, fakeReader(tc.objects), f)
			fileObjects, err := p.Parse(nil, "", true)
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

			p := filesystem.NewRawParser(cmpath.Root{}, fakeReader(tc.objects), f)

			_, err := p.Parse(tc.syncedCRDs, "", true)
			if err == nil {
				t.Fatal("expected error")
			}

			errs := err.Errors()
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
