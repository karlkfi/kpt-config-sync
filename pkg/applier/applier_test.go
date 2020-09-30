package applier

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/syncer/syncertest"
	testingfake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNs1 = "fake-namespace-1"
	testNs2 = "fake-namespace-2"
	testNs3 = "fake-namespace-3"
)

// TestApply verifies that the apply can compare the declared resource from git with
// the previously cached resource and take the right actions.
func TestApply(t *testing.T) {
	var tcs = []struct {
		name string
		// scope is the applier's scope.
		scope declared.Scope
		// the git resource to which the applier syncs the state to.
		declared []ast.FileObject
		// cached is the cache of the set of previously-declared objects.
		cached []ast.FileObject
		// actual is the objects currently on the cluster.
		actual []ast.FileObject
		// expected changes happened to each resource.
		wantEvents []Event
		// expected GVKs returned from apply()
		wantGVKs map[schema.GroupVersionKind]struct{}
		// wantErr is the set of errors we want to occur.
		wantErr status.MultiError
	}{
		{
			name:  "Create Test - if the resource is missing.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				// shall be created.
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			cached: []ast.FileObject{},
			actual: []ast.FileObject{},
			wantEvents: []Event{
				{"Create", testNs1},
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
			},
		},
		{
			name:  "Update Test - if the resource is previously cached.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			cached: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			wantEvents: []Event{
				{"Update", testNs2},
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
			},
		},
		{
			name:     "Delete Test - if the cached resource is not in the upcoming resource",
			scope:    declared.RootReconciler,
			declared: []ast.FileObject{},
			cached: []ast.FileObject{
				fake.Namespace("namespace/"+testNs3, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs3, syncertest.ManagementEnabled),
			},
			wantEvents: []Event{
				{"Delete", testNs3},
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{},
		},
		{
			name:  "CUD Test - all three at once",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			cached: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs3, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
				fake.Namespace("namespace/"+testNs3, syncertest.ManagementEnabled),
			},
			wantEvents: []Event{
				{"Create", testNs1},
				{"Update", testNs2},
				{"Delete", testNs3},
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
			},
		},
		{
			name:  "Ignore Test - if the resource has the configManagement disabled.",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementDisabled),
			},
			actual: []ast.FileObject{},
			// testNs1 is not touched.
			wantEvents: []Event{},
			// We still want to watch Namespaces so we can unmanage the actual.
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
			},
		},
		// We don't need to test every possible path here; we've already done that
		// in diff_test.go. This just ensures we can reach the switch-case branches we expect.
		{
			name:  "Management Conflict Test1 - declared and actual resource managed by root",
			scope: declared.RootReconciler,
			declared: []ast.FileObject{
				fake.Role(core.Name("admin"), syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Role(core.Name("admin"), syncertest.ManagementEnabled, difftest.ManagedByRoot),
			},
			wantEvents: []Event{
				{"Update", "admin"},
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Role(): {},
			},
		},
		{
			name:  "Management Conflict Test2 - declared managed by Namespace, and actual resource managed by root",
			scope: "shipping",
			declared: []ast.FileObject{
				fake.Role(core.Name("admin"), core.Namespace("shipping"),
					syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Role(core.Name("admin"), core.Namespace("shipping"),
					syncertest.ManagementEnabled, difftest.ManagedByRoot),
			},
			wantGVKs: map[schema.GroupVersionKind]struct{}{
				kinds.Role(): {},
			},
			wantErr: ManagementConflictError(fake.Role()),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientForTest(t)
			clientApplier := &FakeApplier{WantActions: tc.wantEvents}
			// Propagate the actual resources.
			for _, actual := range tc.actual {
				if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
					t.Fatal(err)
				}
			}
			previousCache := make(map[core.ID]core.Object)
			for _, cached := range tc.cached {
				previousCache[core.IDOf(cached)] = cached.Object
			}
			var a *Applier
			if tc.scope == declared.RootReconciler {
				a = NewRootApplier(fakeClient, clientApplier)
			} else {
				a = NewNamespaceApplier(fakeClient, clientApplier, tc.scope)
			}
			a.cachedObjects = previousCache
			// Verify.
			gvks, err := a.Apply(context.Background(), filesystem.AsCoreObjects(tc.declared))
			if err != nil || tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("got Apply() = error %v, want error %v", err, tc.wantErr)
				}
				return
			}

			if diff := cmp.Diff(tc.wantGVKs, gvks); diff != "" {
				t.Errorf("Diff of GVK map from Apply(): %s", diff)
			}

			if len(clientApplier.WantActions) == 0 && len(clientApplier.GotActions) == 0 {
				return
			}
			if diff := cmp.Diff(clientApplier.WantActions, clientApplier.GotActions,
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
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
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
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertest.ManagementEnabled),
			},
			want: []Event{{"Update", testNs1}},
		},
		{
			name:     "Delete Test2 - applier refresh cannot delete resources.",
			declared: []ast.FileObject{},
			actual: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertest.ManagementEnabled),
			},
			want: []Event{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := clientForTest(t)
			// Propagate the actual resource to api server
			for _, actual := range tc.actual {
				if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
					t.Fatal(err)
				}
			}
			clientApplier := &FakeApplier{WantActions: tc.want}
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

			if diff := cmp.Diff(clientApplier.WantActions, clientApplier.GotActions,
				cmpopts.SortSlices(func(x, y Event) bool { return x.Action < y.Action }),
				cmpopts.EquateEmpty()); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

type FakeApplier struct {
	WantActions []Event
	GotActions  []Event
	client      client.Client
}

var _ reconcile.Applier = &FakeApplier{}

type Event struct {
	Action string
	Name   string
}

func (a *FakeApplier) Create(_ context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.GotActions = append(a.GotActions, Event{"Create", obj.GetName()})
	return true, nil
}

func (a *FakeApplier) Update(_ context.Context, i, _ *unstructured.Unstructured) (
	bool, status.Error) {
	a.GotActions = append(a.GotActions, Event{"Update", i.GetName()})
	return true, nil
}

func (a *FakeApplier) RemoveNomosMeta(_ context.Context, intent *unstructured.Unstructured) (
	bool, status.Error) {
	a.GotActions = append(a.GotActions, Event{"RemoveNomosMeta",
		intent.GetName()})
	return true, nil
}

func (a *FakeApplier) Delete(_ context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.GotActions = append(a.GotActions, Event{"Delete", obj.GetName()})
	return true, nil
}

func (a *FakeApplier) GetClient() client.Client {
	return a.client
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

func TestSortByScope(t *testing.T) {
	namespaced := diff.Diff{Declared: fake.RoleObject(core.Namespace("shipping"))}
	clustered := diff.Diff{Declared: fake.ClusterRoleObject()}

	testCases := []struct {
		name  string
		left  diff.Diff
		right diff.Diff
		want  bool
	}{
		{
			name:  "both namespaced",
			left:  namespaced,
			right: namespaced,
			want:  false,
		},
		{
			name:  "first namespaced",
			left:  namespaced,
			right: clustered,
			want:  false,
		},
		{
			name:  "second namespaced",
			left:  clustered,
			right: namespaced,
			want:  true,
		},
		{
			name:  "both clustered",
			left:  clustered,
			right: clustered,
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := clusterScopedFirst(tc.left, tc.right)
			if got != tc.want {
				t.Errorf("got clusterScopedFirst() = %t, want %t", got, tc.want)
			}
		})
	}
}
