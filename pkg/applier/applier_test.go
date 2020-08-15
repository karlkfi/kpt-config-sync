package applier

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	testingfake "github.com/google/nomos/pkg/syncer/testing/fake"
	"github.com/google/nomos/pkg/testing/fake"
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
	cases := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		declaredResources []ast.FileObject
		// The previously cached resource.
		actualResources []ast.FileObject
		// expected changes happened to each resource.
		expectedActions []Event
	}{
		{
			name: "Create Test2 - if the resource is missing.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
				// shall be created.
				fake.Namespace("namespace/" + testNs2),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{
				{"Update", testNs1},
				{"Create", testNs2}},
		},
		{
			name: "No-Op - if the resource has the configManagement disabled.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementDisabled),
				fake.Namespace("namespace/" + testNs2),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs2, syncertesting.ManagementEnabled),
			},
			// testNs1 is not touched.
			expectedActions: []Event{{"Update", testNs2}},
		},
		{
			name: "Update Test1 - if the resource is previously cached.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Update", testNs1}},
		},
		{
			name: "Delete Test2 - if the cached resource is not in the upcoming resource",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{
				{"Delete", testNs2},
				{"Update", testNs1}},
		},
	}
	for _, test := range cases {
		fakeClient := clientForTest(t)
		clientApplier := &FakeApplier{ExpectActions: test.expectedActions}
		previousCache := make(map[core.ID]core.Object)
		// Propagate the actual resources.
		for _, actual := range test.actualResources {
			if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
				t.Fatal(err)
			}
			previousCache[core.IDOf(actual)] = actual.Object
		}
		a := NewRootApplier(fakeClient, clientApplier)
		a.cachedObjects = previousCache
		// Verify.
		if err := a.Apply(context.Background(), filesystem.AsCoreObjects(test.declaredResources)); err != nil {
			t.Errorf("test %q failed: %v", test.name, err)
		}
		if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
			return
		}
		if diff := cmp.Diff(clientApplier.ExpectActions, clientApplier.ActualActions,
			cmpopts.SortSlices(func(x, y Event) bool { return x.Action < y.Action })); diff != "" {
			t.Errorf(
				"test %q failed, diff between expected event and actual events: \n%s",
				test.name, diff)
		}
	}
}

// TestRefresh verifies that applier Refresh can keep the state in the API server in sync with
// the git resource in sync.
func TestRefresh(t *testing.T) {
	cases := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		declaredResources []ast.FileObject
		// the API serve resource from which propagates the applier cache.
		actualResource []ast.FileObject
		// expected changes happened to each resource.
		expectedActions []Event
	}{
		{
			name: "Create Test1 - if the declared resource is not in the API server.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actualResource:  []ast.FileObject{},
			expectedActions: []Event{{"Create", testNs1}},
		},
		{
			name: "No-Op - if the declared resource is management disabled changed.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementDisabled),
			},
			actualResource:  []ast.FileObject{},
			expectedActions: []Event{},
		},
		{
			name: "Update Test1 - if the declared resource is in API server.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actualResource: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Update", testNs1}},
		},
		{
			name: "Delete Test2 - if the resource in API server no longer has upcoming resource",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			actualResource: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Delete", testNs2}},
		},
	}
	for _, test := range cases {
		fakeClient := clientForTest(t)
		// Propagate the actual resource to api server
		for _, actual := range test.actualResource {
			if err := fakeClient.Create(context.Background(), actual.Object); err != nil {
				t.Fatal(err)
			}
		}
		clientApplier := &FakeApplier{ExpectActions: test.expectedActions}
		a := NewRootApplier(fakeClient, clientApplier)
		// The cache is used to store the declared git resource. Assuming it is out of sync
		// with the state in the API server.
		a.cachedObjects = make(map[core.ID]core.Object)
		for _, actual := range test.declaredResources {
			a.cachedObjects[core.IDOf(actual)] = actual.Object
		}

		err := a.Refresh(context.Background())
		// Verify.
		if err != nil {
			t.Errorf("test %q failed: %v", test.name, err)
		}
		if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
			return
		}
		if diff := cmp.Diff(clientApplier.ExpectActions, clientApplier.ActualActions,
			cmpopts.SortSlices(func(x, y Event) bool { return x.Action < y.Action })); diff != "" {
			t.Errorf(
				"test %q failed, diff between expected event and actual events: \n%s",
				test.name, diff)
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
