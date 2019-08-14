package filesystem_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	fstesting "github.com/google/nomos/pkg/importer/filesystem/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	importToken = visitortesting.ImportToken
	loadTime    = time.Time{}
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
			expected: namespaceconfig.NewAllConfigs(importToken, loadTime),
		},
		{
			name: "cluster-scoped object",
			objects: []ast.FileObject{
				fake.ClusterRole(),
			},
			expected: func() *namespaceconfig.AllConfigs {
				result := namespaceconfig.NewAllConfigs(importToken, loadTime)
				result.AddClusterResource(fake.ClusterRole().Object)
				result.AddSync(*v1.NewSync(kinds.ClusterRole().GroupKind()))
				return result
			}(),
		},
		{
			name: "preserves Namespace",
			objects: []ast.FileObject{
				fake.Role(object.Namespace("foo")),
				fake.RoleBinding(object.Namespace("bar")),
			},
			expected: func() *namespaceconfig.AllConfigs {
				result := namespaceconfig.NewAllConfigs(importToken, loadTime)
				result.AddNamespaceResource("foo", fake.Role(object.Namespace("foo")).Object)
				result.AddNamespaceResource("bar", fake.RoleBinding(object.Namespace("bar")).Object)
				result.AddSync(*v1.NewSync(kinds.Role().GroupKind()))
				result.AddSync(*v1.NewSync(kinds.RoleBinding().GroupKind()))
				return result
			}(),
		},
		{
			name: "defaults to default Namespace",
			objects: []ast.FileObject{
				fake.Role(),
			},
			expected: func() *namespaceconfig.AllConfigs {
				result := namespaceconfig.NewAllConfigs(importToken, loadTime)
				// Whether the Role specifies metadata.namespace has no impact on the behavior of applying
				// the object.
				result.AddNamespaceResource("default", fake.Role().Object)
				result.AddSync(*v1.NewSync(kinds.Role().GroupKind()))
				return result
			}(),
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
