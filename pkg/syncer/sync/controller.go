package sync

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const syncControllerName = "meta-sync-resources"

var unaryHandler = &handler.EnqueueRequestsFromMapFunc{
	ToRequests: handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: "item"}}}
	}),
}

// AddController adds the Sync controller to the manager.
func AddController(mgr manager.Manager, managerRestartCh chan event.GenericEvent) error {
	dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrapf(err, "failed to create discoveryclient")
	}
	// Set up a meta controller that restarts GenericResource controllers when Syncs change.
	startErrCh := make(chan error)
	clientFactory := func() (client.Client, error) {
		cfg := mgr.GetConfig()
		mapper, err2 := apiutil.NewDiscoveryRESTMapper(cfg)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "failed to create mapper during gc")
		}
		return client.New(cfg, client.Options{
			Scheme: scheme.Scheme,
			Mapper: mapper,
		})
	}
	reconciler, err := NewMetaReconciler(mgr, dc, clientFactory, startErrCh)
	if err != nil {
		return errors.Wrapf(err, "could not create %q reconciler", syncControllerName)
	}

	c, err := controller.New(syncControllerName, mgr, controller.Options{
		Reconciler: reconciler,
	})
	if err != nil {
		return errors.Wrapf(err, "could not create %q controller", syncControllerName)
	}

	// Watch all changes to Syncs.
	if err = c.Watch(&source.Kind{Type: &v1.Sync{}}, unaryHandler); err != nil {
		return errors.Wrapf(err, "could not watch Syncs in the %q controller", syncControllerName)
	}
	// Watch all changes to NamespaceConfigs.
	// There is a corner case, where a user creates a repo with only namespaces in it.
	// In order for the NamespaceConfig reconciler to start reconciling NamespaceConfigs,
	// a Sync needed to be created to cause us to start the NamespaceConfig controller.
	// We watch NamespaceConfigs so we can reconcile namespaces for this specific scenario.
	if err = c.Watch(&source.Kind{Type: &v1.NamespaceConfig{}}, unaryHandler); err != nil {
		return errors.Wrapf(err, "could not watch NamespaceConfigs in the %q controller", syncControllerName)
	}

	// Create a watch for errors when starting the subManager and force a reconciliation.
	managerRestartSource := &source.Channel{Source: managerRestartCh}
	if err = c.Watch(managerRestartSource, &handler.EnqueueRequestForObject{}); err != nil {
		return errors.Wrapf(err, "could not watch manager initialization errors in the %q controller", syncControllerName)
	}

	go func() {
		for {
			startErr := <-startErrCh
			if startErr != nil {
				// subManager could not successfully start, so we must force it to restart next reconcile.
				glog.Errorf("Error starting NamespaceConfig / ClusterConfig controllers, restarting: %v", startErr)

				// Send an event that forces the subManager to restart.
				managerRestartCh <- event.GenericEvent{Meta: &metav1.ObjectMeta{Name: ForceRestart}}
			}
		}
	}()

	return nil
}
