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
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/labeling"
	syncermanager "github.com/google/nomos/pkg/syncer/manager"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/google/nomos/pkg/util/sync"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &MetaReconciler{}

// ClientFactory is a function used for creating new controller-runtime clients.
type ClientFactory func() (client.Client, error)

// MetaReconciler reconciles Syncs. It responds to changes in Syncs and causes genericResourceManager to stop and start
// controllers based on the resources that are presently sync-enabled.
type MetaReconciler struct {
	// client is used to update Sync status fields and finalize Syncs.
	client *syncerclient.Client
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache cache.Cache
	// discoveryClient is used to look up versions on the cluster for the GroupKinds in the Syncs being reconciled.
	discoveryClient discovery.DiscoveryInterface
	// genericResourceManager is the manager for all sync-enabled resources.
	genericResourceManager syncermanager.RestartableManager
	// mgrStartErrCh is used to listen for errors when (re)starting genericResourceManager.
	mgrStartErrCh chan error
	// clientFactory returns a new dynamic client.
	clientFactory ClientFactory
}

// NewMetaReconciler returns a new MetaReconciler that reconciles changes in Syncs.
func NewMetaReconciler(
	client *syncerclient.Client,
	cache cache.Cache,
	cfg *rest.Config,
	dc discovery.DiscoveryInterface,
	clientFactory ClientFactory,
	errCh chan error) (*MetaReconciler, error) {
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, err
	}

	return &MetaReconciler{
		client:                 client,
		cache:                  cache,
		clientFactory:          clientFactory,
		genericResourceManager: syncermanager.NewGenericResourceManager(mgr, cfg),
		discoveryClient:        dc,
		mgrStartErrCh:          errCh,
	}, nil
}

// Reconcile is the Reconcile callback for MetaReconciler.
// It looks at all Syncs in the cluster and restarts the GenericResourceManager if its internal state doesn't match the cluster
// state.
func (r *MetaReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	syncs := &v1alpha1.SyncList{}
	err := r.cache.List(ctx, &client.ListOptions{}, syncs)
	if err != nil {
		panic(errors.Wrap(err, "could not list all Syncs"))
	}

	var toFinalize []*v1alpha1.Sync
	var enabled []*v1alpha1.Sync
	for idx, s := range syncs.Items {
		if s.GetDeletionTimestamp() != nil {
			// Check for finalizer then finalize if needed.
			toFinalize = append(toFinalize, &syncs.Items[idx])
		} else {
			// Anything not pending delete should be enabled in GenericResourceManager.
			enabled = append(enabled, s.DeepCopy())
		}
	}

	sr, err := r.discoveryClient.ServerResources()
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get api groups")
	}
	apirs, err := utildiscovery.NewAPIInfo(sr)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Check if the set of sync-enabled resources has changed,
	// restart the GenericResourceManager to sync the appropriate resources.
	if err := r.genericResourceManager.UpdateSyncResources(enabled, apirs, r.mgrStartErrCh); err != nil {
		r.genericResourceManager.Clear()
		glog.Errorf("Could not start GenericResourceManager: %v", err)
		return reconcile.Result{}, err
	}

	var errBuilder multierror.Builder
	// Finalize Syncs that have not already been finalized.
	for _, tf := range toFinalize {
		// Make sure to delete all Sync-managed resource before finalizing the Sync.
		errBuilder.Add(r.finalizeSync(ctx, tf, apirs))
	}

	// Update status sub-resource for enabled Syncs, if we have not already done so.
	for _, e := range enabled {
		var status v1alpha1.SyncStatus
		for gk := range sync.GroupKinds(e) {
			statusGroupKind := v1alpha1.SyncGroupVersionKindStatus{
				Group:  gk.Group,
				Kind:   gk.Kind,
				Status: v1alpha1.Syncing,
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
		if !reflect.DeepEqual(e.Status, status) {
			updateFn := func(obj runtime.Object) (runtime.Object, error) {
				s := obj.(*v1alpha1.Sync)
				s.Status = status
				return s, nil
			}
			_, err := r.client.UpdateStatus(ctx, e, updateFn)
			errBuilder.Add(errors.Wrap(err, "could not update sync status"))
		}
	}

	bErr := errBuilder.Build()
	if bErr != nil {
		glog.Errorf("Could not reconcile syncs: %v", bErr)
	}

	return reconcile.Result{}, bErr
}

func (r *MetaReconciler) finalizeSync(ctx context.Context, sync *v1alpha1.Sync, apiInfo *utildiscovery.APIInfo) error {
	// Check if Syncer finalizer is present before finalize.
	finalizers := sets.NewString(sync.GetFinalizers()...)
	if !finalizers.Has(v1alpha1.SyncFinalizer) {
		glog.V(2).Infof("Sync %s already finalized", sync.Name)
		return nil
	}

	sync = sync.DeepCopy()
	finalizers.Delete(v1alpha1.SyncFinalizer)
	sync.SetFinalizers(finalizers.UnsortedList())

	glog.Infof("Beginning Sync finalize for %s", sync.Name)
	if err := r.gcResources(ctx, sync, apiInfo); err != nil {
		return err
	}
	return errors.Wrap(r.client.Upsert(ctx, sync), "could not finalize sync pending delete")
}

func (r *MetaReconciler) gcResources(ctx context.Context, sync *v1alpha1.Sync, apiInfo *utildiscovery.APIInfo) error {
	managed := labels.SelectorFromSet(labels.Set{labeling.ResourceManagementKey: labeling.Enabled})

	// It doesn't matter which version we choose when deleting.
	// Deletes to a resource of a particular version affect all versions with the same group and kind.
	gvks := apiInfo.GroupVersionKinds(sync)
	if len(gvks) == 0 {
		glog.Warningf("Could not find a gvk for %s, CRD may have been deleted, skipping garbage collection.", sync.Name)
		return nil
	}
	var gvk schema.GroupVersionKind
	for k := range gvks {
		gvk = k
		break
	}
	// Create a new dynamic client since it's possible that the manager client is reading from the
	// cache.
	cl, err := r.clientFactory()
	if err != nil {
		return errors.Wrapf(err, "failed to create dynamic client during gc")
	}
	gvk.Kind += "List"
	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	if err := cl.List(ctx, &client.ListOptions{LabelSelector: managed}, ul); err != nil {
		return errors.Wrapf(err, "could not list %s resources", gvk)
	}
	errBuilder := &multierror.Builder{}
	for _, u := range ul.Items {
		if err := cl.Delete(ctx, &u); err != nil {
			errBuilder.Add(errors.Wrapf(err, "could not delete %s resource: %v", gvk, u))
		}
	}
	return errBuilder.Build()
}
