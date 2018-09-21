package controller

import (
	"github.com/google/nomos/clientgen/apis"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"fmt"

	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	genericreconcile "github.com/google/nomos/pkg/generic-syncer/reconcile"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AddClusterPolicy adds ClusterPolicy sync controllers to the Manager.
func AddClusterPolicy(mgr manager.Manager, clientSet *apis.Clientset, gvks []schema.GroupVersionKind) error {
	pnc, err := controller.New("clusterpolicy-resources", mgr, controller.Options{
		Reconciler: genericreconcile.NewClusterPolicyReconciler(clientSet, mgr.GetCache()),
	})
	if err != nil {
		return errors.Wrap(err, "could not create ClusterPolicy controller")
	}
	if err = pnc.Watch(&source.Kind{Type: &nomosv1.ClusterPolicy{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrap(err, "could not watch ClusterPolicies in the controller")
	}

	mapToClusterPolicy := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToClusterPolicy),
	}

	// Set up a watch on all cluster-scoped resources defined in Syncs.
	// Look up the corresponding ClusterPolicy for the changed resources.
	// TODO(116662413): Only watch cluster-scoped generic resources.
	for _, gvk := range gvks {
		t := &unstructured.Unstructured{}
		t.SetGroupVersionKind(gvk)
		name := fmt.Sprintf("clusterpolicy-resources-%s", gvk)
		gc, err := controller.New(name, mgr, controller.Options{
			Reconciler: genericreconcile.NewClusterPolicyReconciler(clientSet, mgr.GetCache()),
		})
		if err != nil {
			return errors.Wrapf(err, "could not create %q controller", name)
		}

		if err := gc.Watch(&source.Kind{Type: t}, mapToClusterPolicy); err != nil {
			return errors.Wrapf(err, "could not watch %q in the generic controller", gvk)
		}
	}
	return nil
}

// genericResourceToClusterPolicy maps generic resources being watched,
// to reconciliation requests for the ClusterPolicy potentially managing them.
func genericResourceToClusterPolicy(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// There is only one ClusterPolicy potentially managing generic resources.
			Name: nomosv1.ClusterPolicyName,
		},
	}}
}
