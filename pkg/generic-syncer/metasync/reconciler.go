/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metasync

import (
	"context"

	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	syncermanager "github.com/google/nomos/pkg/generic-syncer/manager"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &MetaReconciler{}

// MetaReconciler reconciles Syncs. It responds to changes in Syncs and causes genericResourceManager to stop and start
// controllers based on the resources that are presently sync-enabled.
type MetaReconciler struct {
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache cache.Cache
	// genericResourceManager is a manager for all sync-enabled resources.
	genericResourceManager *syncermanager.GenericResourceManager
	// mgrStartErrCh is used to listen for errors when (re)starting genericResourceManager.
	mgrStartErrCh chan error
}

// NewMetaReconciler returns a new MetaReconciler that reconciles changes in Syncs.
func NewMetaReconciler(cache cache.Cache, cfg *rest.Config, errCh chan error) (*MetaReconciler, error) {
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, err
	}

	return &MetaReconciler{
		cache:                  cache,
		genericResourceManager: syncermanager.NewGenericResourceManager(mgr, cfg),
		mgrStartErrCh:          errCh,
	}, nil
}

// Reconcile is the Reconcile callback for MetaReconciler.
// It looks at all Syncs in the cluster and restarts the genericResourceManager if its internal state doesn't match the cluster
// state.
func (r *MetaReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	syncs := &nomosv1alpha1.SyncList{}
	// TODO: pass in a valid context
	err := r.cache.List(context.TODO(), &client.ListOptions{}, syncs)
	if err != nil {
		panic(errors.Wrap(err, "could not list all Syncs"))
	}

	if r.genericResourceManager.ResourcesChanged(syncs) {
		if err := r.genericResourceManager.Restart(syncs, r.mgrStartErrCh); err != nil {
			r.genericResourceManager.Clear()
			return reconcile.Result{}, err
		}
	}

	// TODO: When importer intends to delete Syncs, it needs Syncer to finalize the deletion.
	// We should be doing that finalizing here, when necessary.
	// TODO: Update Syncs with their current status.

	return reconcile.Result{}, nil
}
