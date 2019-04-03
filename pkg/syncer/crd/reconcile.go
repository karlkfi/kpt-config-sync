package crd

import (
	"context"
	"time"

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
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const reconcileTimeout = time.Minute * 5

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles CRD resources on the cluster.
// It restarts ClusterPolicy and NamespaceConfig controllers if changes have been made to CustomResourceDefinitions on the
// cluster.
type Reconciler struct {
	// client is used to update the ClusterConfig.Status
	client *syncerclient.Client
	// applier create/patches/deletes CustomResourceDefinitions.
	applier syncerreconcile.Applier
	// cache is a shared cache that is populated by informers in the scheme and used by all controllers / reconcilers in the
	// manager.
	cache    syncercache.GenericCache
	recorder record.EventRecorder
	decoder  *decode.GenericResourceDecoder
	// managerRestartCh is a channel that listens for events to restart the ClusterConfig and NamespaceConfig controllers and
	// their manager.
	managerRestartCh chan event.GenericEvent
}

// NewReconciler returns a new Reconciler.
func NewReconciler(mgr manager.Manager, managerRestartCh chan event.GenericEvent) (
	*Reconciler, error) {
	resourceClient := syncerclient.New(mgr.GetClient())
	applier, err := syncerreconcile.NewApplier(mgr.GetConfig(), resourceClient)
	if err != nil {
		return nil, err
	}

	return &Reconciler{
		client:           resourceClient,
		applier:          applier,
		cache:            syncercache.NewGenericResourceCache(mgr.GetCache()),
		recorder:         mgr.GetRecorder(crdControllerName),
		decoder:          decode.NewGenericResourceDecoder(mgr.GetScheme()),
		managerRestartCh: managerRestartCh,
	}, nil
}

// Reconcile implements Reconciler.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.Name == v1.ClusterConfigName {
		// We handle the non-CRD Cluster Config in the ClusterConfig controller, so don't reconcile it here.
		return reconcile.Result{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()

	if errs := r.reconcile(ctx); errs.HasErrors() {
		err := errs.Build()
		glog.Errorf("Could not reconcile CRD ClusterConfig: %v", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context) status.ErrorBuilder {
	var errBuilder status.ErrorBuilder
	clusterConfig := &v1.ClusterConfig{}
	if err := r.cache.Get(ctx, types.NamespacedName{Name: v1.CRDClusterConfigName}, clusterConfig); err != nil {
		if apierrors.IsNotFound(err) {
			// CRDs may be changing on the cluster, but we don't have any CRD ClusterConfig to reconcile with.
			return status.ErrorBuilder{}
		}
		errBuilder.Add(errors.Wrapf(err, "could not retrieve ClusterConfig %q", v1.CRDClusterConfigName))
		return errBuilder
	}
	clusterConfig.SetGroupVersionKind(kinds.ClusterConfig())

	actual := &apiextensions.CustomResourceDefinitionList{}
	if err := r.cache.List(ctx, &client.ListOptions{}, actual); err != nil {
		panic(errors.Wrap(err, "could not list all CRDs"))
	}

	grs, err := r.decoder.DecodeResources(clusterConfig.Spec.Resources...)
	if err != nil {
		errBuilder.Add(errors.Wrap(err, "could not decode ClusterConfig"))
		return errBuilder
	}

	gvk := kinds.CustomResourceDefinition()
	declaredInstances := grs[gvk]

	var syncErrs []v1.ConfigManagementError
	actualInstances, err := r.cache.UnstructuredList(gvk, "")
	if err != nil {
		errBuilder.Add(status.APIServerWrapf(err, "failed to list from policy controller for %q", gvk))
		syncErrs = append(syncErrs, syncerreconcile.NewConfigManagementError(clusterConfig, err))
		errBuilder.Add(syncerreconcile.SetClusterConfigStatus(ctx, r.client, clusterConfig, syncErrs...))
		return errBuilder
	}

	allDeclaredVersions := syncerreconcile.AllVersionNames(grs, gvk.GroupKind())
	diffs := differ.Diffs(declaredInstances, actualInstances, allDeclaredVersions)
	var reconcileCount int
	for _, diff := range diffs {
		if updated, err := syncerreconcile.HandleDiff(ctx, r.applier, diff, r.recorder); err != nil {
			errBuilder.Add(err)
			syncErrs = append(syncErrs, syncerreconcile.CmesForResourceError(err)...)
		} else if updated {
			reconcileCount++
		}
	}

	if reconcileCount > 0 {
		// We've updated CRDs on the cluster; restart the NamespaceConfig and ClusterConfig controllers.
		r.managerRestartCh <- event.GenericEvent{Meta: &metav1.ObjectMeta{Name: sync.ForceRestart}}
		r.recorder.Eventf(clusterConfig, corev1.EventTypeNormal, "ReconcileComplete",
			"crd cluster config was successfully reconciled: %d changes", reconcileCount)
	}

	if err := syncerreconcile.SetClusterConfigStatus(ctx, r.client, clusterConfig, syncErrs...); err != nil {
		r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update cluster policy status: %v", err)
		errBuilder.Add(err)
	}

	return errBuilder
}
