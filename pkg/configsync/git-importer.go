package configsync

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/dirwatcher"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/policycontroller"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/syncer/controller"
	"github.com/google/nomos/pkg/syncer/meta"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	clusterName       = flag.String("cluster-name", os.Getenv(reconcilermanager.ClusterNameKey), "Cluster name to use for Cluster selection")
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")

	resyncPeriod = flag.Duration(
		"resync_period", time.Minute, "The resync period for the syncer system")
	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
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

	// Normalize policyDirRelative.
	// Some users specify the directory as if the root of the repository is "/".
	// Strip this from the front of the passed directory so behavior is as
	// expected.
	dir := strings.TrimPrefix(*policyDirRelative, "/")

	// Set up controllers.
	if err := filesystem.AddController(*clusterName, mgr, *gitDir,
		dir, *pollPeriod); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	// Set up controllers.
	if err := meta.AddControllers(mgr); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	mgrStopChannel := signals.SetupSignalHandler()
	ctx := StoppableContext(context.Background(), mgrStopChannel)
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

// StoppableContext returns a Context that will be canceled when the given stop
// channel is closed.
func StoppableContext(ctx context.Context, stopChannel <-chan struct{}) context.Context {
	stoppable, cancel := context.WithCancel(ctx)
	go func() {
		<-stopChannel
		cancel()
	}()
	return stoppable
}
