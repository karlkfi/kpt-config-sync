// Controller responsible for importing policies from a Git repo and materializing CRDs
// on the local cluster.
package main

import (
	"flag"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/google/nomos/pkg/api/configmanagement/v1"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	gitDir            = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")
	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")
	pollPeriod        = flag.Duration("poll-period", time.Second*5, "Poll period for checking if --git-dir target directory has changed")
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

	// Set up Scheme for nomos resources.
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("Error adding configmanagement resaources to scheme: %v", err)
	}

	// Set up controllers.
	if err := filesystem.AddController(mgr, *gitDir, *policyDirRelative, *pollPeriod); err != nil {
		glog.Fatalf("Error adding Sync controller: %+v", err)
	}

	// Start the Manager.
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		glog.Fatalf("Error starting controller: %+v", err)
	}

	glog.Info("Exiting")
}
