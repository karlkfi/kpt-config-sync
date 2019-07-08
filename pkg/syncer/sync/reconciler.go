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
	"github.com/google/nomos/pkg/syncer/metrics"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	now           func() metav1.Time
}

// NewMetaReconciler returns a new MetaReconciler that reconciles changes in Syncs.
func NewMetaReconciler(
	mgr manager.Manager,
	dc discovery.DiscoveryInterface,
	clientFactory ClientFactory,
	now func() metav1.Time,
	errCh chan error) (*MetaReconciler, error) {
	sm, err := manager.New(rest.CopyConfig(mgr.GetConfig()), manager.Options{})
	if err != nil {
		return nil, err
	}

	return &MetaReconciler{
		client:          syncerclient.New(mgr.GetClient(), metrics.APICallDuration),
		cache:           mgr.GetCache(),
		clientFactory:   clientFactory,
		subManager:      syncermanager.NewSubManager(sm, syncermanager.NewSyncAwareBuilder(), errCh),
		discoveryClient: dc,
		now:             now,
	}, nil
}

// Reconcile is the Reconcile callback for MetaReconciler.
// It looks at all Syncs in the cluster and restarts the SubManager if its internal state doesn't match the cluster
// state.
func (r *MetaReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	start := r.now()
	metrics.ReconcileEventTimes.WithLabelValues("sync").Set(float64(start.Unix()))

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	err := r.reconcileSyncs(ctx, request)
	metrics.ReconcileDuration.WithLabelValues("sync", metrics.StatusLabel(err)).Observe(time.Since(start.Time).Seconds())

	if err != nil {
		glog.Errorf("Could not reconcile syncs: %v", err)
	}
	return reconcile.Result{}, err
}

func (r *MetaReconciler) reconcileSyncs(ctx context.Context, request reconcile.Request) error {
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
		return errors.Wrapf(err, "failed to get api groups")
	}
	apirs, err := utildiscovery.NewAPIInfo(sr)
	if err != nil {
		return err
	}

	eventTriggeredRestart := restartSubManager(request.Name)
	source := "sync"
	if eventTriggeredRestart {
		// For event triggered forced restarts, we store the source of the restart in the Namespace field.
		source = request.Namespace
	}

	attemptedRestart, err := r.subManager.Restart(apirs.GroupVersionKinds(enabled...), apirs, eventTriggeredRestart)
	if attemptedRestart {
		metrics.ControllerRestarts.WithLabelValues(source).Inc()
	}
	if err != nil {
		glog.Errorf("Could not start SubManager: %v", err)
		return err
	}

	var mErr status.MultiError
	// Finalize Syncs that have not already been finalized.
	for _, tf := range toFinalize {
		// Make sure to delete all Sync-managed resource before finalizing the Sync.
		gvksToFinalize := apirs.GroupVersionKinds(tf)
		mErr = status.Append(mErr, r.finalizeSync(ctx, tf, gvksToFinalize))
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
			mErr = status.Append(mErr, status.APIServerWrapf(err, "could not update sync status"))
		}
	}

	return mErr
}

func (r *MetaReconciler) finalizeSync(ctx context.Context, sync *v1.Sync, gvks map[schema.GroupVersionKind]bool) status.MultiError {
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
	if err := r.gcResources(ctx, sync, gvks); err != nil {
		return err
	}
	err := r.client.Upsert(ctx, sync)
	return status.From(status.APIServerWrapf(err, "could not finalize sync pending delete"))
}

func (r *MetaReconciler) gcResources(ctx context.Context, sync *v1.Sync, gvks map[schema.GroupVersionKind]bool) status.MultiError {
	// It doesn't matter which version we choose when deleting.
	// Deletes to a resource of a particular version affect all versions with the same group and kind.
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

// restartSubManager returns true if the reconcile request indicates that we need to restart all
// controllers that the Sync Controller manages.
func restartSubManager(name string) bool {
	return name == ForceRestart
}
