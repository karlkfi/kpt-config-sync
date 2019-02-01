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
	"reflect"
	"sort"
	"time"

	"github.com/golang/glog"
	nomosv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncermanager "github.com/google/nomos/pkg/syncer/manager"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &MetaReconciler{}

// MetaReconciler reconciles Syncs. It responds to changes in Syncs and causes genericResourceManager to stop and start
// controllers based on the resources that are presently sync-enabled.
type MetaReconciler struct {
	// client is used to update Sync status fields and finalize Syncs.
	client *syncerclient.Client
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache cache.Cache
	// genericResourceManager is the manager for all sync-enabled resources.
	genericResourceManager syncermanager.RestartableManager
	// mgrStartErrCh is used to listen for errors when (re)starting genericResourceManager.
	mgrStartErrCh chan error
}

// NewMetaReconciler returns a new MetaReconciler that reconciles changes in Syncs.
func NewMetaReconciler(client *syncerclient.Client, cache cache.Cache, cfg *rest.Config, errCh chan error) (*MetaReconciler,
	error) {
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, err
	}

	return &MetaReconciler{
		client:                 client,
		cache:                  cache,
		genericResourceManager: syncermanager.NewGenericResourceManager(mgr, cfg),
		mgrStartErrCh:          errCh,
	}, nil
}

// Reconcile is the Reconcile callback for MetaReconciler.
// It looks at all Syncs in the cluster and restarts the GenericResourceManager if its internal state doesn't match the cluster
// state.
func (r *MetaReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	syncs := &nomosv1alpha1.SyncList{}
	err := r.cache.List(ctx, &client.ListOptions{}, syncs)
	if err != nil {
		panic(errors.Wrap(err, "could not list all Syncs"))
	}

	var toFinalize []*nomosv1alpha1.Sync
	var enabled []*nomosv1alpha1.Sync
	for _, s := range syncs.Items {
		if s.GetDeletionTimestamp() != nil {
			// Check if Syncer finalizer is present, before including it in the list of syncs to finalize.
			finalizers := sets.NewString(s.GetFinalizers()...)
			if finalizers.Has(nomosv1alpha1.SyncFinalizer) {
				finalizers.Delete(nomosv1alpha1.SyncFinalizer)
				s.SetFinalizers(finalizers.UnsortedList())
				toFinalize = append(toFinalize, &s)
			}
		} else {
			// Anything not pending delete should be enabled in GenericResourceManager.
			enabled = append(enabled, s.DeepCopy())
		}
	}

	// Check if the set of sync-enabled resources has changed,
	// restart the GenericResourceManager to sync the appropriate resources.
	if err := r.genericResourceManager.UpdateSyncResources(enabled, r.mgrStartErrCh); err != nil {
		r.genericResourceManager.Clear()
		glog.Errorf("Could not start GenericResourceManager: %v", err)
		return reconcile.Result{}, err
	}

	var errBuilder multierror.Builder
	// Finalize Syncs that have not already been finalized.
	for _, tf := range toFinalize {
		errBuilder.Add(errors.Wrap(r.client.Upsert(ctx, tf), "could not finalize sync pending delete"))
	}

	// Update status sub-resource for enabled Syncs, if we have not already done so.
	for _, sync := range enabled {
		var status nomosv1alpha1.SyncStatus
		for gvk := range syncermanager.GroupVersionKinds(sync) {
			statusGroupKind := nomosv1alpha1.SyncGroupVersionKindStatus{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    gvk.Kind,
				Status:  nomosv1alpha1.Syncing,
			}
			status.GroupVersionKinds = append(status.GroupVersionKinds, statusGroupKind)
		}
		// Sort the status fields, so we can compare statuses.
		sort.Slice(status.GroupVersionKinds, func(i, j int) bool {
			lgk, rgk := status.GroupVersionKinds[i], status.GroupVersionKinds[j]
			if lgk.Group < rgk.Group {
				return true
			}
			return lgk.Kind < rgk.Kind
		})
		// Check if status changed before updating.
		if !reflect.DeepEqual(sync.Status, status) {
			updateFn := func(obj runtime.Object) (runtime.Object, error) {
				s := obj.(*nomosv1alpha1.Sync)
				s.Status = status
				return s, nil
			}
			_, err := r.client.UpdateStatus(ctx, sync, updateFn)
			errBuilder.Add(errors.Wrap(err, "could not update sync status"))
		}
	}

	bErr := errBuilder.Build()
	if bErr != nil {
		glog.Errorf("Could not reconcile syncs: %v", bErr)
	}

	return reconcile.Result{}, bErr
}
