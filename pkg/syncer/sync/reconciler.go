/*
Copyright 2018 The CSP Config Management Authors.
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

package sync

import (
	"context"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	syncermanager "github.com/google/nomos/pkg/syncer/manager"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
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

const (
	// ForceRestart is an invalid resource name used to signal that during Reoconcile,
	// the Sync Controller must restart the Sub Manager. Ensuring that the resource name
	// is invalid ensures that we don't accidentally reconcile a resource that causes us
	// to forcefully restart the SubManager.
	ForceRestart     = "@restart"
	reconcileTimeout = time.Minute * 5
)

var _ reconcile.Reconciler = &MetaReconciler{}

// ClientFactory is a function used for creating new controller-runtime clients.
type ClientFactory func() (client.Client, error)

// MetaReconciler reconciles Syncs. It responds to changes in Syncs and causes subManager to stop and start
// controllers based on the resources that are presently sync-enabled.
type MetaReconciler struct {
	// client is used to update Sync status fields and finalize Syncs.
	client *syncerclient.Client
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache cache.Cache
	// discoveryClient is used to look up versions on the cluster for the GroupKinds in the Syncs being reconciled.
	discoveryClient discovery.DiscoveryInterface
	// subManager is responsible for starting/restarting all controllers that depend on Syncs.
	subManager syncermanager.RestartableManager
	// clientFactory returns a new dynamic client.
	clientFactory ClientFactory
}

// NewMetaReconciler returns a new MetaReconciler that reconciles changes in Syncs.
func NewMetaReconciler(
	mgr manager.Manager,
	dc discovery.DiscoveryInterface,
	clientFactory ClientFactory,
	errCh chan error) (*MetaReconciler, error) {
	sm, err := manager.New(rest.CopyConfig(mgr.GetConfig()), manager.Options{})
	if err != nil {
		return nil, err
	}

	return &MetaReconciler{
		client:          syncerclient.New(mgr.GetClient()),
		cache:           mgr.GetCache(),
		clientFactory:   clientFactory,
		subManager:      syncermanager.NewSubManager(sm, syncermanager.NewSyncAwareBuilder(), errCh),
		discoveryClient: dc,
	}, nil
}

// Reconcile is the Reconcile callback for MetaReconciler.
// It looks at all Syncs in the cluster and restarts the SubManager if its internal state doesn't match the cluster
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
			// Anything not pending delete should be enabled in SyncAwareBuilder.
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

	if err := r.subManager.Restart(apirs.GroupVersionKinds(enabled...), apirs, restartSubManager(request)); err != nil {
		glog.Errorf("Could not start SubManager: %v", err)
		return reconcile.Result{}, err
	}

	var errBuilder status.MultiError
	// Finalize Syncs that have not already been finalized.
	for _, tf := range toFinalize {
		// Make sure to delete all Sync-managed resource before finalizing the Sync.
		errBuilder = status.Append(errBuilder, r.finalizeSync(ctx, tf, apirs))
	}

	// Update status sub-resource for enabled Syncs, if we have not already done so.
	for _, sync := range enabled {
		var ss v1.SyncStatus
		ss.Status = v1.Syncing

		// Check if status changed before updating.
		if !reflect.DeepEqual(sync.Status, ss) {
			updateFn := func(obj runtime.Object) (runtime.Object, error) {
				s := obj.(*v1.Sync)
				s.Status = ss
				return s, nil
			}
			sync.SetGroupVersionKind(kinds.Sync())
			_, err := r.client.UpdateStatus(ctx, sync, updateFn)
			errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "could not update sync status"))
		}
	}

	if errBuilder != nil {
		glog.Errorf("Could not reconcile syncs: %v", errBuilder)
	}
	return reconcile.Result{}, errBuilder
}

func (r *MetaReconciler) finalizeSync(ctx context.Context, sync *v1.Sync, apiInfo *utildiscovery.APIInfo) status.MultiError {
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
	err := r.client.Upsert(ctx, sync)
	return status.From(status.APIServerWrapf(err, "could not finalize sync pending delete"))
}

func (r *MetaReconciler) gcResources(ctx context.Context, sync *v1.Sync, apiInfo *utildiscovery.APIInfo) status.MultiError {
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
	var errBuilder status.MultiError
	// Create a new dynamic client since it's possible that the manager client is reading from the
	// cache.
	cl, err := r.clientFactory()
	if err != nil {
		errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "failed to create dynamic client during gc"))
		return errBuilder
	}
	gvk.Kind += "List"
	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(gvk)
	if err := cl.List(ctx, &client.ListOptions{}, ul); err != nil {
		errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "could not list %s resources", gvk))
		return errBuilder
	}
	for _, u := range ul.Items {
		annots := u.GetAnnotations()
		if v, ok := annots[v1.ResourceManagementKey]; !ok || v != v1.ResourceManagementEnabled {
			continue
		}
		if err := cl.Delete(ctx, &u); err != nil {
			errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "could not delete %s resource: %v", gvk, u))
		}
	}
	return errBuilder
}

// restartSubManager returns true if the reconcile request indicates that we need to restart all the controllers that the
// Sync Controller manages.
func restartSubManager(request reconcile.Request) bool {
	return request.Name == ForceRestart
}
