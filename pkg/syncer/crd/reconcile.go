package crd

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/syncer/metrics"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	syncercache "github.com/google/nomos/pkg/syncer/cache"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/syncer/sync"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles CRD resources on the cluster.
// It restarts ClusterConfig and NamespaceConfig controllers if changes have been made to
// CustomResourceDefinitions on the cluster.
type Reconciler struct {
	// client is used to update the ClusterConfig.Status
	client *syncerclient.Client
	// applier create/patches/deletes CustomResourceDefinitions.
	applier syncerreconcile.Applier
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache    syncercache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	now      func() metav1.Time
	// signal is a handle that is used to restart the ClusterConfig and NamespaceConfig controllers and
	// their manager.
	signal sync.RestartSignal
}

// NewReconciler returns a new Reconciler.
func NewReconciler(client *syncerclient.Client, applier syncerreconcile.Applier,
	cache syncercache.GenericCache, recorder record.EventRecorder, decoder decode.Decoder,
	now func() metav1.Time, signal sync.RestartSignal) *Reconciler {
	return &Reconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		now:      now,
		signal:   signal,
	}
}

// Reconcile implements Reconciler.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.Name != v1.CRDClusterConfigName {
		// We only handle the CRD ClusterConfig in this reconciler.
		return reconcile.Result{}, nil
	}

	start := r.now()
	metrics.ReconcileEventTimes.WithLabelValues("crd").Set(float64(start.Unix()))

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	err := r.reconcile(ctx, request.Name)
	metrics.ReconcileDuration.WithLabelValues("crd", metrics.StatusLabel(err)).Observe(time.Since(start.Time).Seconds())

	if err != nil {
		glog.Errorf("Could not reconcile CRD ClusterConfig: %v", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, name string) status.MultiError {
	var mErr status.MultiError

	clusterConfig := &v1.ClusterConfig{}
	if err := r.cache.Get(ctx, types.NamespacedName{Name: name}, clusterConfig); err != nil {
		if apierrors.IsNotFound(err) {
			// CRDs may be changing on the cluster, but we don't have any CRD ClusterConfig to reconcile with.
			return nil
		}
		err = errors.Wrapf(err, "could not retrieve ClusterConfig %q", v1.CRDClusterConfigName)
		glog.Error(err)
		mErr = status.Append(mErr, err)
		return mErr
	}
	clusterConfig.SetGroupVersionKind(kinds.ClusterConfig())

	grs, err := r.decoder.DecodeResources(clusterConfig.Spec.Resources...)
	if err != nil {
		mErr = status.Append(mErr, errors.Wrap(err, "could not decode ClusterConfig"))
		return mErr
	}
	gvk := kinds.CustomResourceDefinition()
	declaredInstances := grs[gvk]
	for _, decl := range declaredInstances {
		syncerreconcile.SyncedAt(decl, clusterConfig.Spec.Token)
	}

	var syncErrs []v1.ConfigManagementError
	actualInstances, err := r.cache.UnstructuredList(gvk, "")
	if err != nil {
		mErr = status.Append(mErr, status.APIServerWrapf(err, "failed to list from config controller for %q", gvk))
		syncErrs = append(syncErrs, syncerreconcile.NewConfigManagementError(clusterConfig, err))
		mErr = status.Append(mErr, syncerreconcile.SetClusterConfigStatus(ctx, r.client, clusterConfig,
			r.now, syncErrs...))
		return mErr
	}

	allDeclaredVersions := syncerreconcile.AllVersionNames(grs, gvk.GroupKind())
	diffs := differ.Diffs(declaredInstances, actualInstances, allDeclaredVersions)
	var reconcileCount int
	for _, diff := range diffs {
		if updated, err := syncerreconcile.HandleDiff(ctx, r.applier, diff, r.recorder); err != nil {
			mErr = status.Append(mErr, err)
			syncErrs = append(syncErrs, err.ToCME())
		} else if updated {
			reconcileCount++
		}
	}

	if reconcileCount > 0 {
		// We've updated CRDs on the cluster; restart the NamespaceConfig and ClusterConfig controllers.
		r.signal.Restart()
		r.recorder.Eventf(clusterConfig, corev1.EventTypeNormal, "ReconcileComplete",
			"crd cluster config was successfully reconciled: %d changes", reconcileCount)
	}

	if err := syncerreconcile.SetClusterConfigStatus(ctx, r.client, clusterConfig, r.now, syncErrs...); err != nil {
		r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update ClusterConfig status: %v", err)
		mErr = status.Append(mErr, err)
	}

	return mErr
}
