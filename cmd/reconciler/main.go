package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/go-logr/glogr"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	ocmetrics "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/log"
	ctrl "sigs.k8s.io/controller-runtime"
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
		"The relative path of the root policy directory within the repo.")

	// Performance tuning flags.
	gitDir = flag.String(flags.gitDir, "/repo/source/rev",
		"The absolute path in the container running the reconciler to the clone of the git repo.")
	repoRootDir = flag.String(flags.repoRootDir, "/repo",
		"The absolute path in the container running the reconciler to the repo root directory.")
	hydratedRootDir = flag.String(flags.hydratedRootDir, "/repo/hydrated",
		"The absolute path in the container running the reconciler to the hydrated root directory.")
	hydratedLinkDir = flag.String("hydrated-link", "rev",
		"The name of (a symlink to) the source directory under --hydrated-root, which contains the hydrated configs")
	fightDetectionThreshold = flag.Float64(
		"fight-detection-threshold", 5.0,
		"The rate of updates per minute to an API Resource at which the Syncer logs warnings about too many updates to the resource.")
	resyncPeriod = flag.Duration("resync-period", time.Hour,
		"Period of time between forced re-syncs from Git (even without a new commit).")
	workers = flag.Int("workers", 1,
		"Number of concurrent remediator workers to run at once.")
	filesystemPollingPeriod = flag.Duration("filesystem-polling-period", controllers.PollingPeriod(reconcilermanager.ReconcilerPollingPeriod, configsync.DefaultReconcilerPollingPeriod),
		"Period of time between checking the filesystem for updates to the source or rendered configs.")

	// Root-Repo-only flags. If set for a Namespace-scoped Reconciler, causes the Reconciler to fail immediately.
	sourceFormat = flag.String(flags.sourceFormat, os.Getenv(filesystem.SourceFormatKey),
		"The format of the repository.")

	debug = flag.Bool("debug", false,
		"Enable debug mode, panicking in many scenarios where normally an InternalError would be logged. "+
			"Do not use in production.")
)

var flags = struct {
	gitDir          string
	repoRootDir     string
	hydratedRootDir string
	clusterName     string
	sourceFormat    string
}{
	repoRootDir:     "repo-root",
	gitDir:          "git-dir",
	hydratedRootDir: "hydrated-root",
	clusterName:     "cluster-name",
	sourceFormat:    reconcilermanager.SourceFormat,
}

func main() {
	flag.Parse()
	log.Setup()
	ctrl.SetLogger(glogr.New())

	if *debug {
		status.EnablePanicOnMisuse()
	}

	// Register the OpenCensus views
	if err := ocmetrics.RegisterReconcilerMetricsViews(); err != nil {
		glog.Fatalf("Failed to register OpenCensus views: %v", err)
	}

	// Register the OC Agent exporter
	oce, err := ocmetrics.RegisterOCAgentExporter()
	if err != nil {
		glog.Fatalf("Failed to register the OC Agent exporter: %v", err)
	}

	defer func() {
		if err := oce.Stop(); err != nil {
			glog.Fatalf("Unable to stop the OC Agent exporter: %v", err)
		}
	}()

	absRepoRoot, err := cmpath.AbsoluteOS(*repoRootDir)
	if err != nil {
		glog.Fatalf("%s must be an absolute path: %v", flags.repoRootDir, err)
	}

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
		RepoRoot:                   absRepoRoot,
		HydratedRoot:               *hydratedRootDir,
		HydratedLink:               *hydratedLinkDir,
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
	reconciler.Run(opts)
}
