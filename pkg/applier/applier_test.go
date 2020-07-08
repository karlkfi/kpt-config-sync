package applier

import (
	"context"
	"reflect"
	"testing"

	syncertesting "github.com/google/nomos/pkg/syncer/testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/status"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	testNs1  = "fake-namespace-1"
	testNs2  = "fake-namespace-2"
	testUID1 = "ab63974a-c0e9-11ea-9a7e-42010a80013f"
	testUID2 = "e41f9d39-c0e9-11ea-9a7e-42010a80013f"
)

func TestApply(t *testing.T) {
	testcases := []struct {
		name              string
		declaredResources []ast.FileObject
		// The previously cached resources
		actualResources []ast.FileObject
		expectedActions []Event
	}{
		{
			name: "Create Test1 -- the very first apply.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, core.UID(testUID1)),
			},
			actualResources: []ast.FileObject{},
			expectedActions: []Event{{"Create", types.UID(testUID1)}},
		},
		{
			name: "Create Test2  -- no cached Test2 resource found.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1, core.UID(testUID1)),
				// shall be created.
				fake.Namespace("namespace/"+testNs2, core.UID(testUID2)),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementEnabled, core.UID(testUID1)),
			},
			expectedActions: []Event{
				{"Update", types.UID(testUID1)},
				{"Create", types.UID(testUID2)}},
		},
		{
			name: "No-Op -- management disabled changed.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementDisabled, core.UID(testUID1)),
			},
			actualResources: []ast.FileObject{},
			expectedActions: []Event{},
		},
		{
			name: "Update Test1.",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementEnabled, core.UID(testUID1)),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementEnabled, core.UID(testUID1)),
			},
			expectedActions: []Event{{"Update", types.UID(testUID1)}},
		},
		{
			name: "Delete Test2",
			declaredResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementEnabled, core.UID(testUID1)),
			},
			actualResources: []ast.FileObject{
				fake.Namespace("namespace/"+testNs1,
					syncertesting.ManagementEnabled, core.UID(testUID1)),
				fake.Namespace("namespace/"+testNs2,
					syncertesting.ManagementEnabled, core.UID(testUID2)),
			},
			expectedActions: []Event{{"Delete", types.UID(testUID2)}},
		},
	}
	for _, test := range testcases {
		clientApplier := &FakeApplier{ExpectActions: test.expectedActions}
		a := New(clientApplier)
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

type FakeApplier struct {
	ExpectActions []Event
	ActualActions []Event
}

type Event struct {
	Action string
	UID    types.UID
}

func (a *FakeApplier) Create(ctx context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Create", obj.GetUID()})
	return true, nil
}

func (a *FakeApplier) Update(ctx context.Context, i, c *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Update", i.GetUID()})
	return true, nil
}

func (a *FakeApplier) RemoveNomosMeta(ctx context.Context, intent *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"RemoveNomosMeta", intent.GetUID()})
	return true, nil
}
func (a *FakeApplier) Delete(ctx context.Context, obj *unstructured.Unstructured) (
	bool, status.Error) {
	a.ActualActions = append(a.ActualActions, Event{"Delete", obj.GetUID()})
	return true, nil
}
