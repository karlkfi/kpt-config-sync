// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"
	"os"
	"time"

	"github.com/google/nomos/pkg/importer/filesystem"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")
	resyncPeriod      = flag.Duration("resync-period", time.Minute, "The resync period for the importer system")
)

func main() {
	flag.Parse()
	log.Setup()

	go service.ServeMetrics()

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

	stopCh := signals.SetupSignalHandler()
	// Set up controllers.
	if err := filesystem.AddController(mgr, *gitDir, *policyDirRelative, *pollPeriod, *resyncPeriod, stopCh); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	// Start the Manager.
	if err := mgr.Start(stopCh); err != nil {
		glog.Fatalf("Error starting controller: %+v", err)
	}

	glog.Info("Exiting")
}
