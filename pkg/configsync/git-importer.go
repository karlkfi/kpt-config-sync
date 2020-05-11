package configsync

import (
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/dirwatcher"
	"github.com/google/nomos/pkg/importer/filesystem"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	clusterName       = flag.String("cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster name to use for Cluster selection")
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")
)

// RunImporter encapsulates the main() logic for the importer.
func RunImporter() {
	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %+v", err)
	}

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatalf("Failed to create manager: %+v", err)
	}

	// Set up Scheme for nomos resources.
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("Error adding configmanagement resources to scheme: %v", err)
	}

	// Set up controllers.
	if err := filesystem.AddController(*clusterName, mgr, *gitDir, *policyDirRelative, *pollPeriod); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	// Start the Manager.
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
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
