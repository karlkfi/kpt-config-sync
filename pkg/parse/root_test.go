// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parse

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	syncertest "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"github.com/pkg/errors"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rootSyncName       = "my-rs"
	rootReconcilerName = "root-reconciler-my-rs"
	nilGitContext      = `{"repo":"","branch":"","rev":""}`
)

type noOpRemediator struct {
	needsUpdate bool
}

func (r *noOpRemediator) ConflictErrors() []status.Error {
	return nil
}

func (r *noOpRemediator) NeedsUpdate() bool {
	return r.needsUpdate
}

func (r *noOpRemediator) ManagementConflict() bool {
	return false
}

func (r *noOpRemediator) UpdateWatches(_ context.Context, _ map[schema.GroupVersionKind]struct{}) status.MultiError {
	r.needsUpdate = false
	return nil
}

func (r *noOpRemediator) Errors() status.MultiError {
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
					core.Label(metadata.ManagedByKey, metadata.ManagedByValue),
					core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion),
					core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementEnabled),
					core.Annotation(metadata.GitContextKey, nilGitContext),
					core.Annotation(metadata.SyncTokenAnnotationKey, ""),
					core.Annotation(metadata.OwningInventoryKey, applier.InventoryID(rootSyncName, configmanagement.ControllerNamespace)),
					core.Annotation(metadata.ResourceIDKey, "_namespace_foo"),
					difftest.ManagedBy(declared.RootReconciler, rootSyncName),
				),
				fake.Role(core.Namespace("foo"),
					core.Label(metadata.ManagedByKey, metadata.ManagedByValue),
					core.Label(metadata.DeclaredVersionLabel, "v1"),
					core.Annotation(metadata.DeclaredFieldsKey, `{"f:metadata":{"f:annotations":{},"f:labels":{}},"f:rules":{}}`),
					core.Annotation(metadata.SourcePathAnnotationKey, "namespaces/foo/role.yaml"),
					core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementEnabled),
					core.Annotation(metadata.GitContextKey, nilGitContext),
					core.Annotation(metadata.SyncTokenAnnotationKey, ""),
					core.Annotation(metadata.OwningInventoryKey, applier.InventoryID(rootSyncName, configmanagement.ControllerNamespace)),
					core.Annotation(metadata.ResourceIDKey, "rbac.authorization.k8s.io_role_foo_default-name"),
					difftest.ManagedBy(declared.RootReconciler, rootSyncName),
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
					syncName:           rootSyncName,
					reconcilerName:     rootReconcilerName,
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObjectV1Beta1(rootSyncName)),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					converter:          converter,
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    &fakeApplier{},
					},
					mux: &sync.Mutex{},
				},
			}
			state := reconcilerState{}
			if err := parseAndUpdate(context.Background(), parser, triggerReimport, &state); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, state.cache.objsToApply, cmpopts.EquateEmpty(), ast.CompareFileObject, cmpopts.SortSlices(sortObjects)); diff != "" {
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
				{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{}}},
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
				{Data: &view.CountData{Value: 2}, Tags: []tag.Tag{{}}},
				{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := &root{
				sourceFormat: filesystem.SourceFormatUnstructured,
				opts: opts{
					parser:             &fakeParser{errors: tc.errors},
					syncName:           rootSyncName,
					reconcilerName:     rootReconcilerName,
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObjectV1Beta1(rootSyncName)),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					updater: updater{
						scope:     declared.RootReconciler,
						resources: &declared.Resources{},
					},
					mux: &sync.Mutex{},
				},
			}
			err := parseAndUpdate(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("parse() should return errors")
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
					syncName:           rootSyncName,
					reconcilerName:     rootReconcilerName,
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObjectV1Beta1(rootSyncName)),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					updater: updater{
						scope:     declared.RootReconciler,
						resources: &declared.Resources{},
					},
					mux: &sync.Mutex{},
				},
			}
			err := parseAndUpdate(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("parse() should return errors")
			}
			if diff := m.ValidateMetrics(metrics.ReconcilerErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func TestRoot_SourceAndSyncReconcilerErrorsMetricValidation(t *testing.T) {
	testCases := []struct {
		name        string
		applyErrors []status.Error
		wantMetrics []*view.Row
	}{
		{
			name: "single reconciler error in sync component",
			applyErrors: []status.Error{
				applier.Error(errors.New("sync error")),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 0}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "parsing"}}},
				{Data: &view.LastValueData{Value: 0}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "source"}}},
				{Data: &view.LastValueData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "sync"}}},
			},
		},
		{
			name: "multiple reconciler errors in sync component",
			applyErrors: []status.Error{
				applier.Error(errors.New("sync error")),
				status.InternalError("internal error"),
			},
			wantMetrics: []*view.Row{
				{Data: &view.LastValueData{Value: 0}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "parsing"}}},
				{Data: &view.LastValueData{Value: 0}, Tags: []tag.Tag{{Key: metrics.KeyComponent, Value: "source"}}},
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
					parser: &fakeParser{},
					updater: updater{
						scope:      declared.RootReconciler,
						resources:  &declared.Resources{},
						remediator: &noOpRemediator{},
						applier:    &fakeApplier{errors: tc.applyErrors},
					},
					syncName:           rootSyncName,
					reconcilerName:     rootReconcilerName,
					client:             syncertest.NewClient(t, runtime.NewScheme(), fake.RootSyncObjectV1Beta1(rootSyncName)),
					discoveryInterface: syncertest.NewDiscoveryClient(kinds.Namespace(), kinds.Role()),
					mux:                &sync.Mutex{},
				},
			}
			err := parseAndUpdate(context.Background(), parser, triggerReimport, &reconcilerState{})
			if err == nil {
				t.Errorf("update() should return errors")
			}
			if diff := m.ValidateMetrics(metrics.ReconcilerErrorsView, tc.wantMetrics); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func sortObjects(left, right client.Object) bool {
	leftID := core.IDOf(left)
	rightID := core.IDOf(right)
	return leftID.String() < rightID.String()
}

type fakeParser struct {
	parse  []ast.FileObject
	errors []status.Error
}

func (p *fakeParser) Parse(_ reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	if p.errors == nil {
		return p.parse, nil
	}
	var errs status.MultiError
	for _, e := range p.errors {
		errs = status.Append(errs, e)
	}
	return nil, errs
}

func (p *fakeParser) ReadClusterRegistryResources(_ reader.FilePaths, _ filesystem.SourceFormat) ([]ast.FileObject, status.MultiError) {
	return nil, nil
}

func (p *fakeParser) ReadClusterNamesFromSelector(_ reader.FilePaths) ([]string, status.MultiError) {
	return nil, nil
}

type fakeApplier struct {
	got    []client.Object
	errors []status.Error
}

func (a *fakeApplier) Apply(_ context.Context, objs []client.Object) (map[schema.GroupVersionKind]struct{}, status.MultiError) {
	if a.errors == nil {
		a.got = objs
		gvks := make(map[schema.GroupVersionKind]struct{})
		for _, obj := range objs {
			gvks[obj.GetObjectKind().GroupVersionKind()] = struct{}{}
		}
		return gvks, nil
	}
	var errs status.MultiError
	for _, e := range a.errors {
		errs = status.Append(errs, e)
	}
	return nil, errs
}

func (a *fakeApplier) Errors() status.MultiError {
	var errs status.MultiError
	for _, e := range a.errors {
		errs = status.Append(errs, e)
	}
	return errs
}

func (a *fakeApplier) Syncing() bool {
	return false
}

func TestSummarizeErrors(t *testing.T) {
	testCases := []struct {
		name                 string
		sourceStatus         v1beta1.GitSourceStatus
		syncStatus           v1beta1.GitSyncStatus
		expectedErrorSources []v1beta1.ErrorSource
		expectedErrorSummary v1beta1.ErrorSummary
	}{
		{
			name:                 "both sourceStatus and syncStatus are empty",
			sourceStatus:         v1beta1.GitSourceStatus{},
			syncStatus:           v1beta1.GitSyncStatus{},
			expectedErrorSources: []v1beta1.ErrorSource{},
			expectedErrorSummary: v1beta1.ErrorSummary{},
		},
		{
			name: "sourceStatus is not empty (no trucation), syncStatus is empty",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus:           v1beta1.GitSyncStatus{},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                2,
				Truncated:                 false,
				ErrorCountAfterTruncation: 2,
			},
		},
		{
			name: "sourceStatus is not empty and trucates errors, syncStatus is empty",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus:           v1beta1.GitSyncStatus{},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                100,
				Truncated:                 true,
				ErrorCountAfterTruncation: 2,
			},
		},
		{
			name:         "sourceStatus is empty, syncStatus is not empty (no trucation)",
			sourceStatus: v1beta1.GitSourceStatus{},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                2,
				Truncated:                 false,
				ErrorCountAfterTruncation: 2,
			},
		},
		{
			name:         "sourceStatus is empty, syncStatus is not empty and trucates errors",
			sourceStatus: v1beta1.GitSourceStatus{},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                100,
				Truncated:                 true,
				ErrorCountAfterTruncation: 2,
			},
		},
		{
			name: "neither sourceStatus nor syncStatus is empty or trucates errors",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError, v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                4,
				Truncated:                 false,
				ErrorCountAfterTruncation: 4,
			},
		},
		{
			name: "neither sourceStatus nor syncStatus is empty, sourceStatus trucates errors",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError, v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                102,
				Truncated:                 true,
				ErrorCountAfterTruncation: 4,
			},
		},
		{
			name: "neither sourceStatus nor syncStatus is empty, syncStatus trucates errors",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                2,
					Truncated:                 false,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},

				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError, v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                102,
				Truncated:                 true,
				ErrorCountAfterTruncation: 4,
			},
		},
		{
			name: "neither sourceStatus nor syncStatus is empty, both trucates errors",
			sourceStatus: v1beta1.GitSourceStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "1021", ErrorMessage: "1021-error-message"},
					{Code: "1022", ErrorMessage: "1022-error-message"},
				},
				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			syncStatus: v1beta1.GitSyncStatus{
				Errors: []v1beta1.ConfigSyncError{
					{Code: "2009", ErrorMessage: "apiserver error"},
					{Code: "2009", ErrorMessage: "webhook error"},
				},

				ErrorSummary: &v1beta1.ErrorSummary{
					TotalCount:                100,
					Truncated:                 true,
					ErrorCountAfterTruncation: 2,
				},
			},
			expectedErrorSources: []v1beta1.ErrorSource{v1beta1.SourceError, v1beta1.SyncError},
			expectedErrorSummary: v1beta1.ErrorSummary{
				TotalCount:                200,
				Truncated:                 true,
				ErrorCountAfterTruncation: 4,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotErrorSources, gotErrorSummary := summarizeErrors(tc.sourceStatus, tc.syncStatus)
			if diff := cmp.Diff(tc.expectedErrorSources, gotErrorSources); diff != "" {
				t.Errorf("summarizeErrors() got %v, expected %v", gotErrorSources, tc.expectedErrorSources)
			}
			if diff := cmp.Diff(tc.expectedErrorSummary, gotErrorSummary); diff != "" {
				t.Errorf("summarizeErrors() got %v, expected %v", gotErrorSummary, tc.expectedErrorSummary)
			}
		})
	}
}
