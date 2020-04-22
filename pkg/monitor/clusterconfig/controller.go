// Package clusterconfig contains the controller for monitoring Nomos ClusterConfigs.
package clusterconfig

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/util/repo"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/monitor/state"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName   = "nomos-monitor-clusterconfig-controller"
	reconcileTimeout = time.Minute * 5
)

var _ reconcile.Reconciler = &reconciler{}

// reconciler responds to changes to ClusterConfigs by updating its ClusterState.
type reconciler struct {
	cache  cache.Cache
	state  *state.ClusterState
	repoCl *repo.Client
}

// Reconcile is the callback for Reconciler.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.Name != v1.ClusterConfigName && request.Name != v1.CRDClusterConfigName {
		glog.Errorf("ClusterConfig has invalid name %q", request.Name)
		// Return nil since we don't want to queue a retry.
		return reconcile.Result{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	cp := &v1.ClusterConfig{}
	err := r.cache.Get(ctx, types.NamespacedName{Name: request.Name}, cp)
	switch {
	case err == nil:
		err = r.state.ProcessClusterConfig(cp)
	case errors.IsNotFound(err):
		r.state.DeleteConfig(request.Name)
		err = nil
	default:
		glog.Errorf("Failed to fetch ClusterConfig for %q.", request.Name)
	}
	if err != nil {
		glog.Errorf("Could not reconcile ClusterConfig %q: %v", request.Name, err)
	}

	if repoObj, err := r.repoCl.GetOrCreateRepo(ctx); err != nil {
		glog.Errorf("Failed to fetch Repo: %v", err)
	} else {
		r.state.ProcessRepo(repoObj)
	}
	return reconcile.Result{}, err
}

// AddController adds a controller to the given manager which reconciles monitoring data for cluster
// configs.
func AddController(mgr manager.Manager, repoCl *repo.Client, cs *state.ClusterState) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: &reconciler{
			cache:  mgr.GetCache(),
			state:  cs,
			repoCl: repoCl,
		},
	})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &v1.ClusterConfig{}}, &handler.EnqueueRequestForObject{})
}
