package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/log"
	"github.com/pkg/errors"
)

var (
	clusterName = flag.String(flags.clusterName, os.Getenv(reconcilermanager.ClusterNameKey),
		"Cluster name to use for Cluster selection")
	scope = flag.String("scope", os.Getenv("SCOPE"),
		"Scope of the reconciler, either a namespace or ':root'.")

	// Git configuration flags. These values originate in the ConfigManagement and
	// configure git-sync to clone the desired repository/reference we want.
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
	filesystemPollingPeriod = flag.Duration("filesystem-polling-period", pollingPeriod(),
		"Period of time between checking the filessystem for udpates to the local Git repository.")

	// Root-Repo-only flags. If set for a Namespace-scoped Reconciler, causes the Reconciler to fail immediately.
	sourceFormat = flag.String(flags.sourceFormat, os.Getenv(filesystem.SourceFormatKey),
		"The format of the repository.")

	debug = flag.Bool("debug", false,
		"Enable debug mode, panicking in many scenarios where normally an InternalError would be logged. "+
			"Do not use in production.")
)

var flags = struct {
	gitDir       string
	clusterName  string
	sourceFormat string
}{
	gitDir:       "git-dir",
	clusterName:  "cluster-name",
	sourceFormat: reconcilermanager.SourceFormat,
}

func main() {
	flag.Parse()
	log.Setup()

	if *debug {
		status.EnablePanicOnMisuse()
	}

	// Register the OpenCensus views
	if err := metrics.RegisterReconcilerMetricsViews(); err != nil {
		glog.Fatalf("Failed to register OpenCensus views: %v", err)
	}

	// Register the Prometheus exporter
	go service.ServePrometheusMetrics(true)

	// Normalize policyDirRelative.
	// Some users specify the directory as if the root of the repository is "/".
	// Strip this from the front of the passed directory so behavior is as
	// expected.
	dir := strings.TrimPrefix(*policyDir, "/")
	relPolicyDir := cmpath.RelativeOS(dir)
	absGitDir, err := cmpath.AbsoluteOS(*gitDir)
	if err != nil {
		glog.Fatalf("%s must be an absolute path: %v", flags.gitDir, err)
	}

	err = declared.ValidateScope(*scope)
	if err != nil {
		glog.Fatal(err)
	}

	dc, err := importer.DefaultCLIOptions.ToDiscoveryClient()
	if err != nil {
		glog.Fatalf("Failed to get DiscoveryClient: %v", err)
	}

	opts := reconciler.Options{
		ClusterName:                *clusterName,
		FightDetectionThreshold:    *fightDetectionThreshold,
		NumWorkers:                 *workers,
		ReconcilerScope:            declared.Scope(*scope),
		ResyncPeriod:               *resyncPeriod,
		FilesystemPollingFrequency: *filesystemPollingPeriod,
		GitRoot:                    absGitDir,
		GitRev:                     *gitRev,
		GitBranch:                  *gitBranch,
		GitRepo:                    *gitRepo,
		PolicyDir:                  relPolicyDir,
		DiscoveryClient:            dc,
	}

	if declared.Scope(*scope) == declared.RootReconciler {
		// Default to "hierarchy" if unset.
		format := filesystem.SourceFormat(*sourceFormat)
		if format == "" {
			format = filesystem.SourceFormatHierarchy
		}

		glog.Info("Starting reconciler for: root")
		opts.RootOptions = &reconciler.RootOptions{
			SourceFormat: format,
		}
	} else {
		glog.Infof("Starting reconciler for: %s", *scope)

		if *sourceFormat != "" {
			glog.Fatalf("Flag %s and Environment variable%q must not be passed to a Namespace reconciler",
				flags.sourceFormat, filesystem.SourceFormatKey)
		}
	}
	reconciler.Run(context.Background(), opts)
}

func pollingPeriod() time.Duration {
	val, present := os.LookupEnv(reconcilermanager.FilesystemPollingPeriod)
	if present {
		pollingFreq, err := time.ParseDuration(val)
		if err != nil {
			panic(errors.Wrapf(err, "failed to parse environment variable %q,"+
				"got value: %v, want err: nil", reconcilermanager.FilesystemPollingPeriod, pollingFreq))
		}
		return pollingFreq
	}
	return v1alpha1.DefaultFilesystemPollingPeriod
}
