package controller

import (
	"context"

	"github.com/google/nomos/pkg/syncer/metrics"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	syncercache "github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const clusterConfigControllerName = "clusterconfig-resources"

// AddClusterConfig adds ClusterConfig sync controllers to the Manager.
func AddClusterConfig(ctx context.Context, mgr manager.Manager, decoder decode.Decoder,
	resourceTypes map[schema.GroupVersionKind]runtime.Object) error {
	genericClient := client.New(mgr.GetClient(), metrics.APICallDuration)
	oa, err := syncerreconcile.OpenAPIResources(mgr.GetConfig())
	if err != nil {
		return err
	}
	applier := syncerreconcile.NewApplier(oa, genericClient)

	cpc, err := k8scontroller.New(clusterConfigControllerName, mgr, k8scontroller.Options{
		Reconciler: syncerreconcile.NewClusterConfigReconciler(
			ctx,
			client.New(mgr.GetClient(), metrics.APICallDuration),
			applier,
			syncercache.NewGenericResourceCache(mgr.GetCache()),
			&CancelFilteringRecorder{mgr.GetEventRecorderFor(clusterConfigControllerName)},
			decoder,
			metav1.Now,
			extractGVKs(resourceTypes),
		),
	})
	if err != nil {
		return errors.Wrapf(err, "could not create %q controller", clusterConfigControllerName)
	}
	if err = cpc.Watch(&source.Kind{Type: &v1.ClusterConfig{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrapf(err, "could not watch ClusterConfigs in the %q controller", clusterConfigControllerName)
	}

	mapToClusterConfig := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(genericResourceToClusterConfig),
	}
	// Set up a watch on all cluster-scoped resources defined in Syncs.
	// Look up the corresponding ClusterConfig for the changed resources.
	for gvk, t := range resourceTypes {
		if err := cpc.Watch(&source.Kind{Type: t}, mapToClusterConfig); err != nil {
			return errors.Wrapf(err, "could not watch %q in the %q controller", gvk, clusterConfigControllerName)
		}
	}
	return nil
}

// genericResourceToClusterConfig maps generic resources being watched,
// to reconciliation requests for the ClusterConfig potentially managing them.
func genericResourceToClusterConfig(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// There is only one ClusterConfig potentially managing generic resources.
			Name: v1.ClusterConfigName,
		},
	}}
}
