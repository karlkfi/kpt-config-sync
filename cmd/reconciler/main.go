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
	scope = flag.String("scope", os.Getenv("SCOPE"),
		"Scope of the reconciler, either a namespace or ':root'.")

	// Git configuration flags. These values originate in the ConfigManagement and
	//configure git-sync to clone the desired repository/reference we want.
	gitRepo = flag.String("git-repo", os.Getenv("GIT_REPO"),
		"The URL of the git repo being synced.")
	gitBranch = flag.String("git-branch", os.Getenv("GIT_BRANCH"),
		"The branch of the git repo being synced.")
	gitRev = flag.String("git-rev", os.Getenv("GIT_REV"),
		"The git reference we're syncing to in the git repo. Could be a specific commit.")
	policyDir = flag.String("policy-dir", os.Getenv("POLICY_DIR"),
		"Relative path of the root policy directory within the repo.")

	// Performance tuning flags.
	gitDir = flag.String(flags.gitDir, "/repo/rev",
		"Absolute path in the container running the Reconciler to the clone of the git repo.")
	fightDetectionThreshold = flag.Float64(
		"fight-detection-threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
	resyncPeriod = flag.Duration("resync-period", time.Hour,
		"Period of time between forced re-syncs from Git (even without a new commit).")
	workers = flag.Int("workers", 1,
		"Number of concurrent remediator workers to run at once.")
	gitPollingPeriod = flag.Duration("git-polling-period", 5*time.Second,
		"Period of time between checking the filessystem for udpates to the local Git repository.")

	// Root-Repo-only flags. If set for a Namespace-scoped Reconciler, causes the Reconciler to fail immediately.
	clusterName = flag.String(flags.clusterName, os.Getenv("CLUSTER_NAME"),
		"Cluster name to use for Cluster selection")
	sourceFormat = flag.String(flags.sourceFormat, os.Getenv(filesystem.SourceFormatKey),
		"The format of the repository.")
)

var flags = struct {
	gitDir       string
	clusterName  string
	sourceFormat string
}{
	gitDir:       "git-dir",
	clusterName:  "cluster-name",
	sourceFormat: "source-format",
}

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

	relPolicyDir := cmpath.RelativeOS(*policyDir)
	absGitDir, err := cmpath.AbsoluteOS(*gitDir)
	if err != nil {
		glog.Fatalf("%s must be an absolute path: %v", flags.gitDir, err)
	}

	err = declared.ValidateScope(*scope)
	if err != nil {
		glog.Fatal(err)
	}

	opts := reconciler.Options{
		FightDetectionThreshold:  *fightDetectionThreshold,
		NumWorkers:               *workers,
		ReconcilerScope:          declared.Scope(*scope),
		ApplierResyncPeriod:      *resyncPeriod,
		GitPollingFrequency:      *gitPollingPeriod,
		GitRoot:                  absGitDir,
		GitRev:                   *gitRev,
		GitBranch:                *gitBranch,
		GitRepo:                  *gitRepo,
		PolicyDir:                relPolicyDir,
		DiscoveryInterfaceGetter: importer.DefaultCLIOptions,
	}
	if declared.Scope(*scope) == declared.RootReconciler {
		glog.Info("Starting reconciler for: root")
		opts.RootOptions = &reconciler.RootOptions{
			ClusterName:  *clusterName,
			SourceFormat: filesystem.SourceFormat(*sourceFormat),
		}
	} else {
		glog.Infof("Starting reconciler for: %s", *scope)

		// TODO(b/167138947): This check needs to be fixed.
		if isFlagPassed(flags.clusterName) || isFlagPassed(flags.sourceFormat) {
			glog.Fatalf("The %s and %s flags must not be passed to a Namespace reconciler",
				flags.clusterName, flags.sourceFormat)
		}
	}
	reconciler.Run(context.Background(), opts)
}
