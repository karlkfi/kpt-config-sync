// Controller responsible for monitoring the state of Nomos resources on the cluster.
package main

import (
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/monitor"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"k8s.io/klog/klogr"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	log.Setup()
	ctrl.SetLogger(klogr.New())

	cfg, err := restconfig.NewRestConfig(restconfig.DefaultTimeout)
	if err != nil {
		klog.Fatalf("Failed to create rest config: %v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		klog.Fatalf("Failed to create manager: %v", err)
	}

	go service.ServeMetrics()

	// Setup all Controllers
	if err := monitor.AddToManager(mgr); err != nil {
		klog.Fatalf("Failed to add controller to manager: %v", err)
	}

	// Start the Manager.
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Fatalf("Error starting controller: %v", err)
	}
}
