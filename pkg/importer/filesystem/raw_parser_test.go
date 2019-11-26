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
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/testoutput"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	importToken = visitortesting.ImportToken
	loadTime    = metav1.Time{}
)

type fakeReader struct {
	fileObjects []ast.FileObject
}

var _ filesystem.Reader = &fakeReader{}

func (r *fakeReader) Read(_ cmpath.Relative, _ bool, _ ...*v1beta1.CustomResourceDefinition) ([]ast.FileObject, status.MultiError) {
	return r.fileObjects, nil
}

func TestRawParser_Parse(t *testing.T) {
	testCases := []struct {
		name     string
		objects  []ast.FileObject
		expected *namespaceconfig.AllConfigs
	}{
		{
			name:     "empty returns empty",
			expected: testoutput.NewAllConfigs(t),
		},
		{
			name: "cluster-scoped object",
			objects: []ast.FileObject{
				fake.ClusterRole(),
			},
			expected: testoutput.NewAllConfigs(t, fake.ClusterRole()),
		},
		{
			name: "preserves Namespace",
			objects: []ast.FileObject{
				fake.Role(core.Namespace("foo")),
				fake.RoleBinding(core.Namespace("bar")),
			},
			expected: testoutput.NewAllConfigs(t,
				fake.Role(core.Namespace("foo")),
				fake.RoleBinding(core.Namespace("bar")),
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := fstesting.NewTestClientGetter(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p := filesystem.NewRawParser(cmpath.Relative{}, &fakeReader{fileObjects: tc.objects}, f)
			result, err := p.Parse(importToken, nil, loadTime, "")
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := fstesting.NewTestClientGetter(t)
			defer func() {
				if err := f.Cleanup(); err != nil {
					t.Fatal(errors.Wrap(err, "could not clean up"))
				}
			}()

			p := filesystem.NewRawParser(cmpath.Relative{}, &fakeReader{fileObjects: tc.objects}, f)

			_, err := p.Parse(importToken, nil, loadTime, "")
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
