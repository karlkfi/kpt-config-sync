package applier

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/nomos/pkg/syncer/reconcile"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
			name: "Create Test2 - if the resource is not previously cached.",
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
			},
			actualResources: []ast.FileObject{},
			expectedActions: []Event{},
		},
		{
			name: "Update Test1 - if the resource is previously cached.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Update", testNs1}},
		},
		{
			name: "Delete Test2 - if the cached resource is not in the upcoming resource",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Delete", testNs2}},
		},
	}
	for _, test := range cases {
		clientApplier := &FakeApplier{ExpectActions: test.expectedActions}
		var items []unstructured.Unstructured
		fakeReader := &FakeReader{listResource: unstructured.UnstructuredList{Items: items}}
		a := New(fakeReader, clientApplier)
		// Propagate the actual resource to the cache.
		for _, actual := range test.actualResources {
			a.cachedResources[core.IDOf(actual)] = actual
		}
		// Verify.
		if err := a.Apply(context.Background(), test.declaredResources); err != nil {
			t.Errorf("test %q failed, unable to prune: %v", test.name, err)
		}

		if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
			return
		}
		if !reflect.DeepEqual(clientApplier.ExpectActions, clientApplier.ActualActions) {
			t.Errorf("test %q failed, was expected to happen %v, \nactual happen %v",
				test.name, clientApplier.ExpectActions, clientApplier.ActualActions)
		}
	}
}

// TestApplyFirstRun verifies that in the very first run, applier can sync its cache with the
// API server and takes the right actions.
func TestApplyFirstRun(t *testing.T) {
	cases := []struct {
		name string
		// the git resource to which the applier syncs the state to.
		declaredResources []ast.FileObject
		// the API serve resource from which propagates the applier cache.
		resourcesStoredInAPIServer []ast.FileObject
		// expected changes happened to each resource.
		expectedActions []Event
	}{
		{
			name: "Create Test1 - if the declared resource is not in the API server.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/" + testNs1),
			},
			resourcesStoredInAPIServer: []ast.FileObject{},
			expectedActions:            []Event{{"Create", testNs1}},
		},
		{
			name: "No-Op - if the declared resource is management disabled changed.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementDisabled),
			},
			resourcesStoredInAPIServer: []ast.FileObject{},
			expectedActions:            []Event{},
		},
		{
			name: "Update Test1 - if the declared resource is in API server.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			resourcesStoredInAPIServer: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Update", testNs1}},
		},
		{
			name: "Delete Test2 - if the resource in API server no longer has upcoming resource",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
			},
			resourcesStoredInAPIServer: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, syncertesting.ManagementEnabled),
				fake.Namespace("namespace/"+testNs2, syncertesting.ManagementEnabled),
			},
			expectedActions: []Event{{"Delete", testNs2}},
		},
	}
	for _, test := range cases {
		clientApplier := &FakeApplier{ExpectActions: test.expectedActions}
		var items []unstructured.Unstructured
		// Propagate the actual resource to api server
		for _, resource := range test.resourcesStoredInAPIServer {
			items = append(items, *unstructuredFn(resource.Object))
		}
		fakeReader := &FakeReader{
			listResource:       unstructured.UnstructuredList{Items: items},
			ExpectedToBeCalled: true}
		a := New(fakeReader, clientApplier)
		// Verify.
		if err := a.Apply(context.Background(), test.declaredResources); err != nil {
			t.Errorf("test %q failed, unable to prune: %v", test.name, err)
		}

		if len(clientApplier.ExpectActions) == 0 && len(clientApplier.ActualActions) == 0 {
			return
		}
		if !reflect.DeepEqual(clientApplier.ExpectActions, clientApplier.ActualActions) {
			t.Errorf("test %q failed, was expected to happen %v, \nactual happen %v",
				test.name, clientApplier.ExpectActions, clientApplier.ActualActions)
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

type FakeReader struct {
	listResource       unstructured.UnstructuredList
	ExpectedToBeCalled bool
}

func (f *FakeReader) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if !f.ExpectedToBeCalled {
		return fmt.Errorf("applier.reader.Get shall not be called")
	}
	return nil
}

func (f *FakeReader) List(ctx context.Context, obj runtime.Object, opts ...client.ListOption) error {
	if f.ExpectedToBeCalled && !f.ExpectedToBeCalled {
		return fmt.Errorf("applier.reader.List shall not be called")
	}
	u, _ := obj.(*unstructured.UnstructuredList)
	u.Items = f.listResource.Items
	return nil
}

var unstructuredFn = func(object core.Object) *unstructured.Unstructured {
	unstructuredObj, _ := reconcile.AsUnstructured(object)
	return unstructuredObj
}
