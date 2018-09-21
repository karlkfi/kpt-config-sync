package reconcile

import (
	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"context"

	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

var _ reconcile.Reconciler = &PolicyNodeReconciler{}

// PolicyNodeReconciler reconciles a PolicyNode object.
type PolicyNodeReconciler struct {
	clientSet *apis.Clientset
	cache     cache.Cache
	scheme    *runtime.Scheme
}

// NewPolicyNodeReconciler returns a new PolicyNodeReconciler.
func NewPolicyNodeReconciler(clientSet *apis.Clientset, cache cache.Cache, scheme *runtime.Scheme) *PolicyNodeReconciler {
	return &PolicyNodeReconciler{
		clientSet: clientSet,
		cache:     cache,
		scheme:    scheme,
	}
}

// Reconcile is the Reconcile callback for PolicyNodeReconciler.
func (r *PolicyNodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	policyNode := &nomosv1.PolicyNode{}
	// TODO: pass in a valid context
	err := r.cache.Get(context.TODO(), request.NamespacedName, policyNode)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not actual, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		glog.Errorf("Could not find %q: %v", request, err)
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	glog.Infof("policy node watched is: %v", policyNode)
	// TODO: reconcile actual namespace-scoped resource with declared policynode state.

	return reconcile.Result{}, nil
}
