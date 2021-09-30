package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/go-logr/glogr"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/hydrate"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/profiler"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/util/log"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	repoRootDir = flag.String("repo-root", "/repo",
		"the absolute path in the container running the hydration to the repo root directory.")

	sourceRootDir = flag.String("source-root", "source",
		"the name of the source root directory under --repo-root.")

	hydratedRootDir = flag.String("hydrated-root", "hydrated",
		"the name of the hydrated root directory under --repo-root.")

	sourceLinkDir = flag.String("source-link", "rev",
		"the name of (a symlink to) the source directory under --source-root, which contains the clone of the git repo.")

	hydratedLinkDir = flag.String("hydrated-link", "rev",
		"the name of (a symlink to) the hydrated directory under --hydrated-root, which contains the hydrated configs")

	syncDir = flag.String("sync-dir", os.Getenv("SYNC_DIR"),
		"Relative path of the root directory within the repo.")

	hydrationPollingPeriodStr = flag.String("polling-period", os.Getenv(reconcilermanager.HydrationPollingPeriod),
		"Period of time between checking the filesystem for rendering the DRY configs.")

	// rehydratePeriod sets the hydration-controller to re-run the hydration process
	// periodically when errors happen. It retries on both transient errors and permanent errors.
	// Other ways to trigger the hydration process are:
	// - push a new commit
	// - delete the done file from the hydration-controller.
	rehydratePeriod = flag.Duration("rehydrate-period", 30*time.Minute,
		"Period of time between rehydrating on errors.")

	scope = flag.String("scope", os.Getenv("SCOPE"),
		"Scope of the hydration controller, either a namespace or ':root'.")
)

func main() {
	flag.Parse()
	log.Setup()
	profiler.Service()
	ctrl.SetLogger(glogr.New())

	absRepoRootDir, err := cmpath.AbsoluteOS(*repoRootDir)
	if err != nil {
		glog.Fatalf("--repo-root must be an absolute path: %v", err)
	}
	absSourceRootDir := absRepoRootDir.Join(cmpath.RelativeSlash(*sourceRootDir))
	absHydratedRootDir := absRepoRootDir.Join(cmpath.RelativeSlash(*hydratedRootDir))
	absDonePath := absRepoRootDir.Join(cmpath.RelativeSlash(hydrate.DoneFile))

	// Normalize syncDirRelative.
	// Some users specify the directory as if the root of the repository is "/".
	// Strip this from the front of the passed directory so behavior is as
	// expected.
	dir := strings.TrimPrefix(*syncDir, "/")
	relSyncDir := cmpath.RelativeOS(dir)

	hydrationPollingPeriod, err := time.ParseDuration(*hydrationPollingPeriodStr)
	if err != nil {
		glog.Fatalf("Failed to get hydration polling period: %v", err)
	}

	var reconcilerName string
	if declared.Scope(*scope) == declared.RootReconciler {
		reconcilerName = reconciler.RootSyncName
	} else {
		reconcilerName = reconciler.RepoSyncName(*scope)
	}

	hydrator := &hydrate.Hydrator{
		DonePath:           absDonePath,
		SourceRoot:         absSourceRootDir,
		HydratedRoot:       absHydratedRootDir,
		SourceLink:         *sourceLinkDir,
		HydratedLink:       *hydratedLinkDir,
		SyncDir:            relSyncDir,
		PollingFrequency:   hydrationPollingPeriod,
		RehydrateFrequency: *rehydratePeriod,
		ReconcilerName:     reconcilerName,
	}

	hydrator.Run(context.Background())
}
