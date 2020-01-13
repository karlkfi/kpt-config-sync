package policycontroller

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/policycontroller/constraint"
	"github.com/google/nomos/pkg/util/watch"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// crdReconciler handles CRD reconcile events and also implements constraint
// Watcher interface.
type crdReconciler struct {
	ctx    context.Context
	client client.Client
	// mgr is the RestartableManager that responds to changes in what kinds of
	// Constraints should be watched.
	mgr watch.RestartableManager
	// crdKinds is a map of CRD name to the kind of resource it defines.
	crdKinds map[string]string
	// constraintKinds is a map of resource kind to boolean indicator if it is
	// established (aka discovery knows about).
	constraintKinds map[string]bool
}

// newReconciler returns a crdReconciler that able to restart the given Manager
// whenever the set of watched Constraint kinds changes.
func newReconciler(ctx context.Context, mgr manager.Manager) (*crdReconciler, error) {
	rm, err := watch.NewManager(mgr, &builder{})
	if err != nil {
		return nil, err
	}
	return &crdReconciler{
		ctx,
		mgr.GetClient(),
		rm,
		map[string]string{},
		map[string]bool{},
	}, nil
}

// Reconcile handles Requests from the PolicyController CRD controller. This may
// update the reconciler's map of established kinds. If this results in a net
// change to which constraint kinds are both watched and established, then the
// RestartableManager will be restarted.
func (c *crdReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	crd := &v1beta1.CustomResourceDefinition{}
	if err := c.client.Get(c.ctx, request.NamespacedName, crd); err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("Error getting CustomResourceDefinition %q: %v", request.NamespacedName, err)
			return reconcile.Result{}, err
		}

		glog.Infof("CustomResourceDefinition %q was deleted", request.NamespacedName)
		if kind, ok := c.crdKinds[request.NamespacedName.String()]; ok {
			delete(c.constraintKinds, kind)
		}
		return reconcile.Result{}, nil
	}

	// We only need to process CRDs that define a PolicyController Constraint.
	if !constraint.MatchesGroup(crd) {
		return reconcile.Result{}, nil
	}

	kind := crd.Spec.Names.Kind
	c.crdKinds[request.NamespacedName.String()] = kind
	c.constraintKinds[kind] = isEstablished(crd)

	_, err := c.mgr.Restart(c.establishedConstraints(), false)
	return reconcile.Result{}, err
}

// isEstablished returns true if the given CRD is established on the cluster,
// which indicates if discovery knows about it yet. For more info see
// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#create-a-customresourcedefinition
func isEstablished(crd *v1beta1.CustomResourceDefinition) bool {
	for _, condition := range crd.Status.Conditions {
		if condition.Type == v1beta1.Established {
			return condition.Status == v1beta1.ConditionTrue
		}
	}
	return false
}

// establishedConstraints returns a map of GVKs for all constraints that have a
// corresponding CRD which is established on the cluster.
func (c *crdReconciler) establishedConstraints() map[schema.GroupVersionKind]bool {
	gvks := map[schema.GroupVersionKind]bool{}
	for kind, established := range c.constraintKinds {
		if established {
			gvks[constraint.GVK(kind)] = true
		}
	}
	return gvks
}
