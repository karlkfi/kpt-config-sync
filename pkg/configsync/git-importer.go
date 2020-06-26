package configsync

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/dirwatcher"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/policycontroller"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/meta"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	clusterName       = flag.String("cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster name to use for Cluster selection")
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")

	resyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")
	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
	enableRemediator = flag.Bool("enable-remediator", false, "enable remediator behavior")
)

// RunImporter encapsulates the main() logic for the importer.
func RunImporter() {
	reconcile.SetFightThreshold(*fightDetectionThreshold)

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %+v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{
		SyncPeriod: resyncPeriod,
	})
	if err != nil {
		glog.Fatalf("Failed to create manager: %+v", err)
	}

	// Set up Scheme for nomos resources.
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("Error adding configmanagement resources to scheme: %v", err)
	}

	decls := declaredresources.NewDeclaredResources()
	genericClient := client.New(mgr.GetClient(), metrics.APICallDuration)
	applier, err := reconcile.NewApplier(mgr.GetConfig(), genericClient)
	if err != nil {
		glog.Fatalf("Instantiating Applier for Remediator: %v", err)
	}

	// Construct the Remediator. This is only directly talked to by the importer
	// controller (defined in package filesystem). The full remediation process is
	// launched in a goroutine when the remediator is constructed.
	var r remediator.Interface
	if *enableRemediator {
		r = remediator.New(mgr.GetClient(), applier, decls)
	} else {
		r = &remediator.NoOp{}
	}

	// Set up controllers.
	if err := filesystem.AddController(*clusterName, mgr, *gitDir,
		*policyDirRelative, *pollPeriod, r); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
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

	glog.Info("Exiting")
}

// DirWatcher watches the filesystem of a given directory until a shutdown signal is received.
func DirWatcher(dir string, period time.Duration) {
	if dir == "" {
		return
	}
	watcher := dirwatcher.NewWatcher(dir)
	watcher.Watch(period, signals.SetupSignalHandler())
}

func stoppableContext(stopChannel <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stopChannel
		cancel()
	}()
	return ctx
}
