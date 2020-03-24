package crd

import (
	"context"
	"reflect"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	syncercache "github.com/google/nomos/pkg/syncer/cache"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/syncer/sync"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// allCrds tracks the entire set of CRDs on the API server.
	// We need to restart the syncer if a CRD is Created/Updated/Deleted since
	// this will change the overall set of resources that the syncer can
	// be handling (in this case gatekeeper will add CRDs to the cluster and we
	// need to restart in order to have the syncer work properly).
	allCrds map[schema.GroupVersionKind]struct{}
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

func (r *Reconciler) listCrds(ctx context.Context) ([]v1beta1.CustomResourceDefinition, error) {
	crdList := &v1beta1.CustomResourceDefinitionList{}
	if err := r.client.List(ctx, crdList, &client.ListOptions{}); err != nil {
		return nil, err
	}
	return crdList.Items, nil
}

func (r *Reconciler) toCrdSet(crds []v1beta1.CustomResourceDefinition) map[schema.GroupVersionKind]struct{} {
	allCRDs := map[schema.GroupVersionKind]struct{}{}
	for _, crd := range crds {
		crdGk := schema.GroupKind{
			Group: crd.Spec.Group,
			Kind:  crd.Spec.Names.Kind,
		}

		if crd.Spec.Version != "" {
			allCRDs[crdGk.WithVersion(crd.Spec.Version)] = struct{}{}
		}
		for _, ver := range crd.Spec.Versions {
			allCRDs[crdGk.WithVersion(ver.Name)] = struct{}{}
		}
	}
	if len(allCRDs) == 0 {
		return nil
	}
	return allCRDs
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

	grs, err := r.decoder.DecodeResources(clusterConfig.Spec.Resources)
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
	actualInstances, err := r.cache.UnstructuredList(ctx, gvk, "")
	//r.cache.List()
	if err != nil {
		mErr = status.Append(mErr, status.APIServerErrorf(err, "failed to list from config controller for %q", gvk))
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
			// TODO(b/152322972): Add unit tests for diff type logic.
			if diff.Type() != differ.Update {
				// We don't need to restart if an existing CRD was updated.
				reconcileCount++
			}
		}
	}

	var needRestart bool
	if reconcileCount > 0 {
		needRestart = true
		// We've updated CRDs on the cluster; restart the NamespaceConfig and ClusterConfig controllers.
		r.recorder.Eventf(clusterConfig, corev1.EventTypeNormal, "ReconcileComplete",
			"crd cluster config was successfully reconciled: %d changes", reconcileCount)
		glog.Info("Triggering restart due to repo CRD change")
	}

	crdList, err := r.listCrds(ctx)
	if err != nil {
		mErr = status.Append(mErr, err)
	} else {
		allCrds := r.toCrdSet(crdList)
		if !reflect.DeepEqual(r.allCrds, allCrds) {
			needRestart = true
			r.recorder.Eventf(clusterConfig, corev1.EventTypeNormal, "CRDChange",
				"crds changed on the cluster restarting syncer controllers")
			glog.Info("Triggering restart due to external CRD change")
		}
		r.allCrds = allCrds
	}

	if needRestart {
		r.signal.Restart("crd")
	}

	if err := syncerreconcile.SetClusterConfigStatus(ctx, r.client, clusterConfig, r.now, syncErrs...); err != nil {
		r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update ClusterConfig status: %v", err)
		mErr = status.Append(mErr, err)
	}

	return mErr
}
