package parse

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/status"
	syncertest "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const nilGitContext = `{"repo":"","branch":"","rev":""}`

func TestRoot_Parse(t *testing.T) {
	testCases := []struct {
		name   string
		format filesystem.SourceFormat
		parsed []core.Object
		want   []core.Object
	}{
		{
			name:   "no objects",
			format: filesystem.SourceFormatUnstructured,
		},
		{
			name:   "implicit namespace if unstructured",
			format: filesystem.SourceFormatUnstructured,
			parsed: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.NamespaceObject("foo",
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					difftest.ManagedByRoot,
				),
				fake.RoleObject(core.Namespace("foo"),
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					difftest.ManagedByRoot,
				),
			},
		},
		// This state technically can't happen as we'd throw an error earlier,
		// but this ensures that if there's a bug in the Parser implementation
		// that we won't erroneously create an implicit Namespace.
		{
			name:   "no implicit namespace if hierarchy",
			format: filesystem.SourceFormatHierarchy,
			parsed: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.RoleObject(core.Namespace("foo"),
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					difftest.ManagedByRoot,
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &fakeRemediator{}
			parser := &root{
				sourceFormat: tc.format,
				opts: opts{
					parser: &fakeParser{parse: tc.parsed},
					updater: updater{
						remediator: r,
						applier:    noOpApplier{},
					},
					client: syncertest.NewClient(t, runtime.NewScheme(),
						fake.RootSyncObject()),
				},
			}

			fakeAbs, err := cmpath.AbsoluteOS("/fake")
			if err != nil {
				t.Fatal(err)
			}

			err = parser.Parse(context.Background(), &gitState{policyDir: fakeAbs})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, r.got, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}

type fakeRemediator struct {
	got []core.Object
}

func (r *fakeRemediator) Update(objects []core.Object) (map[schema.GroupVersionKind]bool, status.MultiError) {
	r.got = objects
	return nil, nil
}

type fakeParser struct {
	parse []core.Object
}

func (p *fakeParser) Parse(clusterName string,
	enableAPIServerChecks bool,
	getSyncedCRDs filesystem.GetSyncedCRDs,
	policyDir cmpath.Absolute,
	files []cmpath.Absolute,
) ([]core.Object, status.MultiError) {
	return p.parse, nil
}

func (p *fakeParser) ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	return nil
}

type noOpApplier struct {
	applier.Interface
}

func (a noOpApplier) Apply(ctx context.Context, watched map[schema.GroupVersionKind]bool, objs []core.Object) status.MultiError {
	return nil
}

func TestSortByScope(t *testing.T) {
	testCases := []struct {
		name string
		objs []core.Object
		want []core.Object
	}{
		{
			name: "Empty list",
			objs: []core.Object{},
			want: []core.Object{},
		},
		{
			name: "Needs sorting",
			objs: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
			want: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Already sorted",
			objs: []core.Object{
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.NamespaceObject("foo"),
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Only namespace scoped",
			objs: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
			want: []core.Object{
				fake.RoleObject(core.Namespace("foo")),
				fake.ResourceQuotaObject(core.Namespace("foo")),
			},
		},
		{
			name: "Only cluster scoped",
			objs: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
			want: []core.Object{
				fake.ClusterRoleObject(),
				fake.NamespaceObject("foo"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]core.Object, len(tc.objs))
			copy(objs, tc.objs)
			sortByScope(objs)
			if diff := cmp.Diff(objs, tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}
