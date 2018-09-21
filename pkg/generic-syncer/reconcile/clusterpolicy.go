package reconcile

import (
	"context"

	"github.com/golang/glog"
	"github.com/google/nomos/clientgen/apis"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &ClusterPolicyReconciler{}

// ClusterPolicyReconciler reconciles a ClusterPolicy object.
type ClusterPolicyReconciler struct {
	clientSet *apis.Clientset
	cache     cache.Cache
}

// NewClusterPolicyReconciler returns a new ClusterPolicyReconciler.
func NewClusterPolicyReconciler(clientSet *apis.Clientset, cache cache.Cache) *ClusterPolicyReconciler {
	return &ClusterPolicyReconciler{
		clientSet: clientSet,
		cache:     cache,
	}
}

// Reconcile is the Reconcile callback for ClusterPolicyReconciler.
func (r *ClusterPolicyReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	clusterPolicy := &nomosv1.ClusterPolicy{}
	// TODO: pass in a valid context
	err := r.cache.Get(context.TODO(), request.NamespacedName, clusterPolicy)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not actual, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	glog.Infof("clusterpolicy watched is: %v", clusterPolicy)
	// TODO: reconcile actual cluster-scoped resource with declared clusterpolicy state.

	return reconcile.Result{}, nil
}
