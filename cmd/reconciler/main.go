package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
)

var (
	fightDetectionThreshold = flag.Float64(
		"fight_detection_threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")

	numWorkers = flag.Int("num_workers", 1, "Number of concurrent remediator workers to run at once.")

	reconcilerScope = flag.String("reconciler-scope", os.Getenv("RECONCILER_SCOPE"), "Scope of the reconciler (either a namespace or ':root').")

	resyncPeriod = flag.Duration("resync_period", time.Hour, "Period of time between forced re-syncs from Git (even without a new commit).")

	gitDir = flag.String("git-dir", "/repo/rev", "Absolute path to the git repo")

	policyDirRelative = flag.String("policy-dir", os.Getenv("POLICY_DIR"), "Relative path of root policy directory in the repo")

	gitPollingPeriod = flag.Duration("git-polling-period", 5*time.Second, "Period of time between checking the filessystem for udpates to the local Git repository.")

	// Root-Repo-only flags. If set for Namespace repos, has no effect.
	clusterName = flag.String("cluster-name", os.Getenv("CLUSTER_NAME"), "Cluster name to use for Cluster selection")

	sourceFormat = flag.String("source-format", os.Getenv(filesystem.SourceFormatKey), "The format of the repository.")
)

const (
	clusterNameFlag  = "cluster-name"
	sourceFormatFlag = "source-format"
)

func isFlagPassed(name string) bool {
	passed := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			passed = true
		}
	})
	return passed
}

func main() {
	flag.Parse()
	log.Setup()

	go service.ServeMetrics()

	opts := reconciler.Options{
		FightDetectionThreshold:  *fightDetectionThreshold,
		NumWorkers:               *numWorkers,
		ReconcilerScope:          *reconcilerScope,
		ApplierResyncPeriod:      *resyncPeriod,
		GitPollingFrequency:      *gitPollingPeriod,
		GitRoot:                  *gitDir,
		PolicyDir:                cmpath.RelativeOS(*policyDirRelative),
		DiscoveryInterfaceGetter: importer.DefaultCLIOptions,
	}
	if *reconcilerScope == declared.RootReconciler {
		opts.RootOptions = &reconciler.RootOptions{
			ClusterName:  *clusterName,
			SourceFormat: filesystem.SourceFormat(*sourceFormat),
		}
	} else if isFlagPassed(clusterNameFlag) || isFlagPassed(sourceFormatFlag) {
		glog.Fatalf("The %s and %s flags must not be passed to a Namespace reconciler",
			clusterNameFlag, sourceFormatFlag)
	}
	reconciler.Run(context.Background(), opts)
}
