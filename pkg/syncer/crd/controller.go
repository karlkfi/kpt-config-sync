package crd

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/metrics"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/syncer/sync"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const crdControllerName = "crd-resources"

// AddCRDController adds the CRD controller to the Manager.
func AddCRDController(mgr manager.Manager, signal sync.RestartSignal) error {
	if err := v1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	resourceClient := syncerclient.New(mgr.GetClient(), metrics.APICallDuration)
	applier, err := syncerreconcile.NewApplier(mgr.GetConfig(), resourceClient)
	if err != nil {
		return err
	}

	reconciler := newReconciler(
		resourceClient,
		applier,
		mgr.GetCache(),
		mgr.GetEventRecorderFor(crdControllerName),
		decode.NewGenericResourceDecoder(mgr.GetScheme()),
		metav1.Now,
		signal,
	)

	cpc, err := k8scontroller.New(crdControllerName, mgr, k8scontroller.Options{
		Reconciler: reconciler,
	})
	if err != nil {
		return errors.Wrapf(err, "could not create %q controller", crdControllerName)
	}

	if err = cpc.Watch(&source.Kind{Type: &v1.ClusterConfig{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrapf(err, "could not watch ClusterConfigs in the %q controller", crdControllerName)
	}

	mapToClusterConfig := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(crdToClusterConfig),
	}
	if err = cpc.Watch(&source.Kind{Type: &v1beta1.CustomResourceDefinition{}}, mapToClusterConfig); err != nil {
		return errors.Wrapf(err, "could not watch CustomResourceDefinitions in the %q controller", crdControllerName)
	}

	return nil
}

// genericResourceToClusterConfig maps generic resources being watched,
// to reconciliation requests for the ClusterConfig potentially managing them.
func crdToClusterConfig(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			// There is only one ClusterConfig potentially managing generic resources.
			Name: v1.CRDClusterConfigName,
		},
	}}
}
