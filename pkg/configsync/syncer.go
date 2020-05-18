package configsync

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/policycontroller"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/meta"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	resyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")
	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
)

// RunSyncer encapsulates the main() logic for the syncer.
func RunSyncer() {
	reconcile.SetFightThreshold(*fightDetectionThreshold)

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %+v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{SyncPeriod: resyncPeriod})
	if err != nil {
		glog.Fatalf("Failed to create manager: %+v", err)
	}

	// Set up controllers.
	if err := meta.AddControllers(mgr); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	mgrStopChannel := signals.SetupSignalHandler()
	ctx := stoppableContext(mgrStopChannel)
	if err := controller.AddRepoStatus(ctx, mgr); err != nil {
		glog.Fatalf("Error adding RepoStatus controller: %+v", err)
	}

	if err := policycontroller.AddControllers(ctx, mgr); err != nil {
		glog.Fatalf("Error adding PolicyController controller: %+v", err)
	}

	// Start the Manager.
	if err := mgr.Start(mgrStopChannel); err != nil {
		glog.Fatalf("Error starting controller: %+v", err)
	}
}

func stoppableContext(stopChannel <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopChannel
		cancel()
	}()
	return ctx
}
