package parse

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/status"
	syncertest "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/vet"
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
			a := &fakeApplier{}
			parser := &root{
				sourceFormat: tc.format,
				opts: opts{
					parser: &fakeParser{parse: tc.parsed},
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    a,
					},
					client: syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
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

			if diff := cmp.Diff(tc.want, a.got, cmpopts.EquateEmpty(), cmpopts.SortSlices(sortObjects)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func sortObjects(left, right core.Object) bool {
	leftID := core.IDOf(left)
	rightID := core.IDOf(right)
	return leftID.String() < rightID.String()
}

type fakeParser struct {
	parse []core.Object
}

func (p *fakeParser) Parse(clusterName string, enableAPIServerChecks bool, addCachedAPIResources vet.AddCachedAPIResourcesFn, getSyncedCRDs filesystem.GetSyncedCRDs, filePaths filesystem.FilePaths) ([]core.Object, status.MultiError) {
	return p.parse, nil
}

func (p *fakeParser) ReadClusterRegistryResources(filePaths filesystem.FilePaths) []ast.FileObject {
	return nil
}

type fakeApplier struct {
	got []core.Object
}

func (a *fakeApplier) Apply(ctx context.Context, objs []core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	a.got = objs
	gvks := make(map[schema.GroupVersionKind]struct{})
	for _, obj := range objs {
		gvks[obj.GroupVersionKind()] = struct{}{}
	}
	return gvks, nil
}
