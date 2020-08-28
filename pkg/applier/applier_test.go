package applier

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/syncertest"
	testingfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	testNs1 = "fake-namespace-1"
	testNs2 = "fake-namespace-2"
)

// TestApply verifies that the apply can compare the declared resource from git with
// the previously cached resource and take the right actions.
func TestApply(t *testing.T) {
	tcs := []struct {
		name string
		// scope is the applier's scope.
		scope declared.Scope
		// the git resource to which the applier syncs the state to.
		declared []ast.FileObject
		// The previously cached resource.
		actual []ast.FileObject
		// expected changes happened to each resource.
		want []Event
		// wantErr is the set of errors we want to occur.
		wantErr status.MultiError
	}{
		{
			name:  "Create Test2 - if the resource is missing.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				// shall be created.
				fake.Namespace("namespace/" + testNs2),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Event{
				{"Update", testNs1},
				{"Create", testNs2}},
		},
		{
			name:  "No-Op - if the resource has the configManagement disabled.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementDisabled),
				fake.Namespace("namespace/" + testNs2),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			// testNs1 is not touched.
			want: []Event{{"Update", testNs2}},
		},
		{
			name:  "Update Test1 - if the resource is previously cached.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Event{{"Update", testNs1}},
		},
		{
			name:  "Delete Test2 - if the cached resource is not in the upcoming resource",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Event{
				{"Delete", testNs2},
				{"Update", testNs1}},
		},
		// We don't need to test every possible path here; we've already done that
		// in diff_test.go. This just ensures we can reach the switch-case branches we expect.
		{
			name:  "Management Conflict Test1 - declared and actual resource managed by root",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Role(core.Name("admin")),
			},
			actual: []ast.FileObject{
				fake.Role(core.Name("admin"), syncertest.ManagementEnabled, difftest.ManagedByRoot),
			},
			want: []Event{
				{"Update", "admin"}},
		},
		{
			name:  "Management Conflict Test2 - declared managed by Namespace, and actual resource managed by root",
			scope: "shipping",
			declared: []ast.FileObject{
				fake.Role(core.Name("admin"), core.Namespace("shipping")),
			},
			actual: []ast.FileObject{
				fake.Role(core.Name("admin"), core.Namespace("shipping"),
					syncertest.ManagementEnabled, difftest.ManagedByRoot),
			},
			wantErr: ManagementConflictError(fake.Role()),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientForTest(t)
			clientApplier := &FakeApplier{ExpectActions: tc.want}
			previousCache := make(map[core.ID]core.Object)
			// Propagate the actual resources.
			for _, actual := range tc.actual {
				if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
					t.Fatal(err)
				}
				previousCache[core.IDOf(actual)] = actual.Object
			}
			var a *Applier
			if tc.scope == declared.RootReconciler {
				a = NewRootApplier(fakeClient, clientApplier)
			} else {
				a = NewNamespaceApplier(fakeClient, clientApplier, tc.scope)
			}
			a.cachedObjects = previousCache
			// Verify.
			err := a.Apply(context.Background(), filesystem.AsCoreObjects(tc.declared))
			if err != nil || tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("got Apply() = error %v, want error %v", err, tc.wantErr)
				}
				return
			}

			if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
				return
			}
			if diff := cmp.Diff(clientApplier.ExpectActions, clientApplier.ActualActions,
				cmpopts.SortSlices(func(x, y Event) bool { return x.Action < y.Action })); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

// TestRefresh verifies that applier Refresh can keep the state in the API server in sync with
// the git resource in sync.
func TestRefresh(t *testing.T) {
	tcs := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		declared []ast.FileObject
		// the API serve resource from which propagates the applier cache.
		actual []ast.FileObject
		// expected changes happened to each resource.
		want []Event
	}{
		{
			name: "Create Test1 - if the declared resource is not in the API server.",
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actual: []ast.FileObject{},
			want:   []Event{{"Create", testNs1}},
		},
		{
			name: "No-Op - if the declared resource is management disabled changed.",
			declared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementDisabled),
			},
			actual: []ast.FileObject{},
			want:   []Event{},
		},
		{
			name: "Update Test1 - if the declared resource is in API server.",
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Event{{"Update", testNs1}},
		},
		{
			name: "Delete Test2 - if the resource in API server no longer has upcoming resource",
			declared: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Event{{"Delete", testNs2}},
		},
	}
	for _, tc := range tcs {
		fakeClient := clientForTest(t)
		// Propagate the actual resource to api server
		for _, actual := range tc.actual {
			if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
				t.Fatal(err)
			}
		}
		clientApplier := &FakeApplier{ExpectActions: tc.want}
		a := NewRootApplier(fakeClient, clientApplier)
		// The cache is used to store the declared git resource. Assuming it is out of sync
		// with the state in the API server.
		a.cachedObjects = make(map[core.ID]core.Object)
		for _, actual := range tc.declared {
			a.cachedObjects[core.IDOf(actual)] = actual.Object
		}

		err := a.Refresh(context.Background())
		// Verify.
		if err != nil {
			t.Error(err)
		}
		if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
			return
		}
		if diff := cmp.Diff(clientApplier.ExpectActions, clientApplier.ActualActions,
			cmpopts.SortSlices(func(x, y Event) bool { return x.Action < y.Action })); diff != "" {
			t.Errorf(diff)
		}
	}
}

type FakeApplier struct {
	ExpectActions []Event
	ActualActions []Event
}

type Event struct {
	Action string
	Name   string
}

func (a *FakeApplier) Create(ctx context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Create", obj.GetName()})
	return true, nil
}

func (a *FakeApplier) Update(ctx context.Context, i, c *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Update", i.GetName()})
	return true, nil
}

func (a *FakeApplier) RemoveNomosMeta(ctx context.Context, intent *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"RemoveNomosMeta",
		intent.GetName()})
	return true, nil
}
func (a *FakeApplier) Delete(ctx context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Delete", obj.GetName()})
	return true, nil
}

func clientForTest(t *testing.T) *testingfake.Client {
	t.Helper()
	s := runtime.NewScheme()
	err := corev1.AddToScheme(s)
	if err != nil {
		t.Fatal(err)
	}

	return testingfake.NewClient(t, s)
}
