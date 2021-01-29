package parse

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	syncertest "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"github.com/google/nomos/pkg/vet"
	"github.com/pkg/errors"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const nilGitContext = `{"repo":"","branch":"","rev":""}`

type noOpRemediator struct {
	needsUpdate bool
}

func (r *noOpRemediator) NeedsUpdate() bool {
	return r.needsUpdate
}

func (r *noOpRemediator) ManagementConflict() bool {
	return false
}

func (r *noOpRemediator) UpdateWatches(ctx context.Context, gvkMap map[schema.GroupVersionKind]struct{}) status.MultiError {
	r.needsUpdate = false
	return nil
}

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
					core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID(configmanagement.ControllerNamespace)),
					difftest.ManagedByRoot,
				),
				fake.RoleObject(core.Namespace("foo"),
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID(configmanagement.ControllerNamespace)),
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
					core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID(configmanagement.ControllerNamespace)),
					difftest.ManagedByRoot,
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := &fakeApplier{}

			fakeAbs, err := cmpath.AbsoluteOS("/fake")
			if err != nil {
				t.Fatal(err)
			}

			parser := &root{
				sourceFormat: tc.format,
				opts: opts{
					parser: &fakeParser{parse: tc.parsed},
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    a,
						cache: cache{
							git: gitState{
								policyDir: fakeAbs,
							},
						},
					},
					client: syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
				},
			}
			err = parse(context.Background(), parser)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, a.got, cmpopts.EquateEmpty(), cmpopts.SortSlices(sortObjects)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestRoot_ParseErrorsMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		errors      []status.Error
		wantMetrics []*view.Row
	}{
		{
			name: "single parse error",
			errors: []status.Error{
				status.InternalError("internal error"),
			},
			wantMetrics: []*view.Row{
				{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyErrorCode, Value: status.InternalErrorCode}}},
			},
		},
		{
			name: "multiple parse errors",
			errors: []status.Error{
				status.InternalError("internal error"),
				status.SourceError.Sprintf("source error").Build(),
				status.InternalError("another internal error"),
			},
			wantMetrics: []*view.Row{
				{Data: &view.CountData{Value: 2}, Tags: []tag.Tag{{Key: metrics.KeyErrorCode, Value: status.InternalErrorCode}}},
				{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyErrorCode, Value: status.SourceErrorCode}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testmetrics.RegisterMetrics(metrics.ParseErrorsView)
			a := &fakeApplier{}

			fakeAbs, err := cmpath.AbsoluteOS("/fake")
			if err != nil {
				t.Fatal(err)
			}

			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					parser: &fakeParser{errors: tc.errors},
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    a,
						cache: cache{
							git: gitState{
								policyDir: fakeAbs,
							},
						},
					},
					client: syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
				},
			}
			_ = parse(context.Background(), parser)
			if diff := m.ValidateMetrics(metrics.ParseErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestRoot_ReconcilerErrorsMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		parseErrors []status.Error
		applyErrors []status.Error
		wantMetrics []*view.Row
	}{
		{
			name: "single reconciler error in source component",
			parseErrors: []status.Error{
				status.SourceError.Sprintf("source error").Build(),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "source"}}},
			},
		},
		{
			name: "multiple reconciler errors in source component",
			parseErrors: []status.Error{
				status.SourceError.Sprintf("source error").Build(),
				status.InternalError("internal error"),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 2}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "source"}}},
			},
		},
		{
			name: "single reconciler error in sync component",
			applyErrors: []status.Error{
				applier.FailedToListResources(errors.New("sync error")),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "sync"}}},
			},
		},
		{
			name: "multiple reconciler errors in sync component",
			applyErrors: []status.Error{
				applier.FailedToListResources(errors.New("sync error")),
				status.InternalError("internal error"),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 2}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "sync"}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testmetrics.RegisterMetrics(metrics.ReconcilerErrorsView)

			fakeAbs, err := cmpath.AbsoluteOS("/fake")
			if err != nil {
				t.Fatal(err)
			}

			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					parser: &fakeParser{errors: tc.parseErrors},
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    &fakeApplier{errors: tc.applyErrors},
						cache: cache{
							git: gitState{
								policyDir: fakeAbs,
							},
						},
					},
					client: syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
				},
			}
			_ = parse(context.Background(), parser)
			if diff := m.ValidateMetrics(metrics.ReconcilerErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
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
	parse  []core.Object
	errors []status.Error
}

func (p *fakeParser) Parse(clusterName string, enableAPIServerChecks bool, addCachedAPIResources vet.AddCachedAPIResourcesFn, getSyncedCRDs filesystem.GetSyncedCRDs, filePaths reader.FilePaths) ([]core.Object, status.MultiError) {
	if p.errors == nil {
		return p.parse, nil
	}
	var errs status.MultiError
	for _, e := range p.errors {
		errs = status.Append(errs, e)
	}
	return nil, errs
}

func (p *fakeParser) ReadClusterRegistryResources(filePaths reader.FilePaths) []ast.FileObject {
	return nil
}

type fakeApplier struct {
	got    []core.Object
	errors []status.Error
}

func (a *fakeApplier) Apply(ctx context.Context, objs []core.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	if a.errors == nil {
		a.got = objs
		gvks := make(map[schema.GroupVersionKind]struct{})
		for _, obj := range objs {
			gvks[obj.GroupVersionKind()] = struct{}{}
		}
		return gvks, nil
	}
	var errs status.MultiError
	for _, e := range a.errors {
		errs = status.Append(errs, e)
	}
	return nil, errs
}
