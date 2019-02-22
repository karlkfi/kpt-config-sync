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
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncermanager "github.com/google/nomos/pkg/syncer/manager"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	syncs := &v1.SyncList{}
	err := r.cache.List(ctx, &client.ListOptions{}, syncs)
	if err != nil {
		panic(errors.Wrap(err, "could not list all Syncs"))
	}

	var toFinalize []*v1.Sync
	var enabled []*v1.Sync
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
	for _, sync := range enabled {
		var status v1.SyncStatus
		status.Status = v1.Syncing

		// Check if status changed before updating.
		if !reflect.DeepEqual(sync.Status, status) {
			updateFn := func(obj runtime.Object) (runtime.Object, error) {
				s := obj.(*v1.Sync)
				s.Status = status
				return s, nil
			}
			sync.SetGroupVersionKind(kinds.Sync())
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

func (r *MetaReconciler) finalizeSync(ctx context.Context, sync *v1.Sync, apiInfo *utildiscovery.APIInfo) error {
	var newFinalizers []string
	var needsFinalize bool
	for _, f := range sync.Finalizers {
		if f == v1.SyncFinalizer {
			needsFinalize = true
		} else {
			newFinalizers = append(newFinalizers, f)
		}
	}

	// Check if Syncer finalizer is present before finalize.
	if !needsFinalize {
		glog.V(2).Infof("Sync %s already finalized", sync.Name)
		return nil
	}

	sync = sync.DeepCopy()
	sync.Finalizers = newFinalizers
	glog.Infof("Beginning Sync finalize for %s", sync.Name)
	if err := r.gcResources(ctx, sync, apiInfo); err != nil {
		return err
	}
	return errors.Wrap(r.client.Upsert(ctx, sync), "could not finalize sync pending delete")
}

func (r *MetaReconciler) gcResources(ctx context.Context, sync *v1.Sync, apiInfo *utildiscovery.APIInfo) error {
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
	if err := cl.List(ctx, &client.ListOptions{}, ul); err != nil {
		return errors.Wrapf(err, "could not list %s resources", gvk)
	}
	errBuilder := &multierror.Builder{}
	for _, u := range ul.Items {
		annots := u.GetAnnotations()
		if v, ok := annots[v1.ResourceManagementKey]; !ok || v != v1.ResourceManagementValue {
			continue
		}
		if err := cl.Delete(ctx, &u); err != nil {
			errBuilder.Add(errors.Wrapf(err, "could not delete %s resource: %v", gvk, u))
		}
	}
	return errBuilder.Build()
}
