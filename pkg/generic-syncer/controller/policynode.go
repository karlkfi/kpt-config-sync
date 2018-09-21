package controller

import (
	"fmt"

	"github.com/google/nomos/clientgen/apis"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	genericreconcile "github.com/google/nomos/pkg/generic-syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AddPolicyNode adds PolicyNode sync controllers to the Manager.
func AddPolicyNode(mgr manager.Manager, clientSet *apis.Clientset, gvks []schema.GroupVersionKind) error {
	pnc, err := controller.New("policynode-resources", mgr, controller.Options{
		Reconciler: genericreconcile.NewPolicyNodeReconciler(clientSet, mgr.GetCache(), mgr.GetScheme()),
	})
	if err != nil {
		return errors.Wrap(err, "could not create policynode controller")
	}
	if err = pnc.Watch(&source.Kind{Type: &nomosv1.PolicyNode{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrap(err, "could not watch PolicyNodes in the controller")
	}

	maptoPolicyNode := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToPolicyNode),
	}

	// Set up a watch on all namespace-scoped resources defined in Syncs.
	// Look up the corresponding PolicyNode for the changed resources.
	// TODO(116662413): Only watch namespace-scoped generic resources.
	for _, gvk := range gvks {
		t := &unstructured.Unstructured{}
		t.SetGroupVersionKind(gvk)
		name := fmt.Sprintf("policynode-resources-%s", gvk)
		gc, err := controller.New(name, mgr, controller.Options{
			Reconciler: genericreconcile.NewPolicyNodeReconciler(clientSet, mgr.GetCache(), mgr.GetScheme()),
		})
		if err != nil {
			return errors.Wrapf(err, "could not create %q controller", name)
		}

		if err := gc.Watch(&source.Kind{Type: t}, maptoPolicyNode); err != nil {
			return errors.Wrapf(err, "could not watch %q in the generic controller", gvk)
		}
	}
	return nil
}

// genericResourceToPolicyNode maps generic resources being watched,
// to reconciliation requests for the PolicyNode potentially managing them.
func genericResourceToPolicyNode(o handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// The namespace of the generic resource is the name of the policy node potentially managing it.
			Name: o.Meta.GetNamespace(),
		},
	}}
}
