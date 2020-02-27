package policycontroller

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/nomos/pkg/policycontroller/constraint"
	"github.com/google/nomos/pkg/util/watch"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestConstraintGVKs(t *testing.T) {
	cm := &clientMock{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	thr := &throttler{make(chan map[schema.GroupVersionKind]bool)}
	go thr.start(ctx, &restartableManagerStub{})

	cr := &crdReconciler{
		context.Background(),
		cm,
		thr,
		map[string]schema.GroupVersionKind{},
		map[schema.GroupVersionKind]bool{},
	}

	// Verify the initial empty case
	gvks := cr.establishedConstraints()
	if len(gvks) != 0 {
		t.Errorf("want empty GVK map; got %v", gvks)
	}

	// Create a FooConstraint that is not yet established.
	cm.nextGet = constraintCRD("FooConstraint", false)
	cr.Reconcile(request("foo"))
	gvks = cr.establishedConstraints()
	if len(gvks) != 0 {
		t.Errorf("want empty GVK map; got %v", gvks)
	}

	// Create a random CRD that is established (but should be ignored).
	cm.nextGet = randomCRD("Anvil", true)
	cr.Reconcile(request("anvil"))
	gvks = cr.establishedConstraints()
	if len(gvks) != 0 {
		t.Errorf("want empty GVK map; got %v", gvks)
	}

	// Create a BarConstraint that is established.
	cm.nextGet = constraintCRD("BarConstraint", true)
	cr.Reconcile(request("bar"))
	gvks = cr.establishedConstraints()
	if len(gvks) != 1 || !gvks[constraint.GVK("BarConstraint")] {
		t.Errorf("want BarConstraint; got %v", gvks)
	}

	// Update FooConstraint to be established along with BarConstraint.
	cm.nextGet = constraintCRD("FooConstraint", true)
	cr.Reconcile(request("foo"))
	gvks = cr.establishedConstraints()
	if len(gvks) != 2 || !gvks[constraint.GVK("FooConstraint")] || !gvks[constraint.GVK("BarConstraint")] {
		t.Errorf("want FooConstraint, BarConstraint; got %v", gvks)
	}

	// Delete BarConstraint from the cluster.
	cm.nextErr = errors.NewNotFound(schema.GroupResource{}, "bar")
	cr.Reconcile(request("bar"))
	gvks = cr.establishedConstraints()
	if len(gvks) != 1 || !gvks[constraint.GVK("FooConstraint")] {
		t.Errorf("want FooConstraint; got %v", gvks)
	}
}

func request(name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{Name: name},
	}
}

func constraintCRD(kind string, isEstablished bool) *v1beta1.CustomResourceDefinition {
	crd := randomCRD(kind, isEstablished)
	crd.Spec.Group = constraint.GVK(kind).Group
	return crd
}

func randomCRD(kind string, isEstablished bool) *v1beta1.CustomResourceDefinition {
	established := v1beta1.ConditionTrue
	if !isEstablished {
		established = v1beta1.ConditionFalse
	}

	return &v1beta1.CustomResourceDefinition{
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: "somethingsomething",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind: kind,
			},
		},
		Status: v1beta1.CustomResourceDefinitionStatus{
			Conditions: []v1beta1.CustomResourceDefinitionCondition{
				{
					Type:   v1beta1.Established,
					Status: established,
				},
			},
		},
	}
}

type clientMock struct {
	client.Client

	nextGet runtime.Object
	nextErr error
}

func (c *clientMock) Get(_ context.Context, _ client.ObjectKey, obj runtime.Object) error {
	if c.nextErr != nil {
		err := c.nextErr
		c.nextErr = nil
		return err
	}

	outVal := reflect.ValueOf(obj)
	reflect.Indirect(outVal).Set(reflect.Indirect(reflect.ValueOf(c.nextGet)))
	c.nextGet = nil
	return nil
}

type restartableManagerStub struct{}

var _ watch.RestartableManager = &restartableManagerStub{}

func (r restartableManagerStub) Restart(_ map[schema.GroupVersionKind]bool, _ bool) (bool, error) {
	return false, nil
}
