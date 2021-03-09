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
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	syncertest "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
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
		parsed []ast.FileObject
		want   []ast.FileObject
	}{
		{
			name:   "no objects",
			format: filesystem.SourceFormatUnstructured,
		},
		{
			name:   "implicit namespace if unstructured",
			format: filesystem.SourceFormatUnstructured,
			parsed: []ast.FileObject{
				fake.Role(core.Namespace("foo")),
			},
			want: []ast.FileObject{
				fake.UnstructuredAtPath(kinds.Namespace(),
					"",
					core.Name("foo"),
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID(configmanagement.ControllerNamespace)),
					difftest.ManagedByRoot,
				),
				fake.Role(core.Namespace("foo"),
					core.Label(v1.ManagedByKey, v1.ManagedByValue),
					core.Label(configuration.DeclaredVersionLabel, "v1"),
					core.Annotation(v1alpha1.DeclaredFieldsKey, `{"f:rules":{}}`),
					core.Annotation(v1.SourcePathAnnotationKey, "namespaces/foo/role.yaml"),
					core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
					core.Annotation(v1alpha1.GitContextKey, nilGitContext),
					core.Annotation(v1.SyncTokenAnnotationKey, ""),
					core.Annotation(kptapplier.OwningInventoryKey, kptapplier.InventoryID(configmanagement.ControllerNamespace)),
					difftest.ManagedByRoot,
				),
			},
		},
	}

	converter, err := declared.ValueConverterForTest()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := &root{
				sourceFormat: tc.format,
				opts: opts{
					parser:             &fakeParser{parse: tc.parsed},
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					converter:          converter,
					updater: updater{
						scope:     declared.RootReconciler,
						resources: &declared.Resources{},
					},
				},
			}
			state := reconcilerState{}
			if err := parse(context.Background(), parser, triggerReimport, &state); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, state.cache.parserResult, cmpopts.EquateEmpty(), ast.CompareFileObject, cmpopts.SortSlices(sortObjects)); diff != "" {
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
			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					parser:             &fakeParser{errors: tc.errors},
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					updater: updater{
						scope:     declared.RootReconciler,
						resources: &declared.Resources{},
					},
				},
			}
			err := parse(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("parse() should return errors")
			}
			if diff := m.ValidateMetrics(metrics.ParseErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestRoot_SourceReconcilerErrorsMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		parseErrors []status.Error
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := testmetrics.RegisterMetrics(metrics.ReconcilerErrorsView)

			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					parser:             &fakeParser{errors: tc.parseErrors},
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					updater: updater{
						scope:     declared.RootReconciler,
						resources: &declared.Resources{},
					},
				},
			}
			err := parse(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("parse() should return errors")
			}
			if diff := m.ValidateMetrics(metrics.ReconcilerErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestRoot_SyncReconcilerErrorsMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		applyErrors []status.Error
		wantMetrics []*view.Row
	}{
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

			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    &fakeApplier{errors: tc.applyErrors},
					},
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObject()),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
				},
			}
			err := update(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("update() should return errors")
			}
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
	parse  []ast.FileObject
	errors []status.Error
}

func (p *fakeParser) Parse(filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	if p.errors == nil {
		return p.parse, nil
	}
	var errs status.MultiError
	for _, e := range p.errors {
		errs = status.Append(errs, e)
	}
	return nil, errs
}

func (p *fakeParser) ReadClusterRegistryResources(filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	return nil, nil
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
