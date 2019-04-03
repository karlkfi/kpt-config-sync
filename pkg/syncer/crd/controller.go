package crd

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const crdControllerName = "crd-resources"

// AddCRDController adds the CRD controller to the Manager.
func AddCRDController(mgr manager.Manager, managerRestartCh chan event.GenericEvent) error {
	if err := v1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	reconciler, err := NewReconciler(mgr, managerRestartCh)
	if err != nil {
		return errors.Wrapf(err, "could not create %q reconciler", crdControllerName)
	}

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
