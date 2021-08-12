package reconciler

import (
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/remediator/watch"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Options contains the settings for a reconciler process.
type Options struct {
	// ClusterName is the name of the cluster we are parsing configuration for.
	ClusterName string
	// FightDetectionThreshold is the rate of updates per minute to an API
	// Resource at which the reconciler will log warnings about too many updates
	// to the resource.
	FightDetectionThreshold float64
	// NumWorkers is the number of concurrent remediator workers to run at once.
	// Each worker pulls resources off of the work queue and remediates them one
	// at a time.
	NumWorkers int
	// ReconcilerScope is the scope of resources which the reconciler will manage.
	// Currently this can either be a namespace or the root scope which allows a
	// cluster admin to manage the entire cluster.
	//
	// At most one Reconciler may have a given value for Scope on a cluster. More
	// than one results in undefined behavior.
	ReconcilerScope declared.Scope
	// ResyncPeriod is the period of time between forced re-sync from Git (even
	// without a new commit).
	ResyncPeriod time.Duration
	// FilesystemPollingFrequency is how often to check the local git repository for
	// changes.
	FilesystemPollingFrequency time.Duration
	// GitRoot is the absolute path to the Git repository.
	// Usually contains a symlink that must be resolved every time before parsing.
	GitRoot cmpath.Absolute
	// HydratedRoot is the absolute path to the hydrated configs.
	// If hydration is not performed, it will be an empty path.
	HydratedRoot string
	// RepoRoot is the absolute path to the parent directory of GitRoot and HydratedRoot.
	RepoRoot cmpath.Absolute
	// HydratedLink is the relative path to the hydrated root.
	// It is a symlink that links to the hydrated configs under the hydrated root dir.
	HydratedLink string
	// GitRev is the git revision being synced.
	GitRev string
	// GitBranch is the git branch being synced.
	GitBranch string
	// GitRepo is the git repo being synced.
	GitRepo string
	// PolicyDir is the relative path to the policies within the Git repository.
	PolicyDir cmpath.Relative
	// DiscoveryClient is used to read types and schemas from the API server.
	DiscoveryClient discovery.DiscoveryInterface
	// RootOptions is the set of options to fill in if this is configuring the
	// Root reconciler.
	// Unset for Namespace repositories.
	*RootOptions
}

// RootOptions are the options specific to parsing Root repositories.
type RootOptions struct {
	// SourceFormat is how the Root repository is structured.
	SourceFormat filesystem.SourceFormat
}

// Run configures and starts the various components of a reconciler process.
func Run(opts Options) {
	reconcile.SetFightThreshold(opts.FightDetectionThreshold)

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig(restconfig.DefaultTimeout)
	if err != nil {
		glog.Fatalf("failed to create rest config: %v", err)
	}

	s := scheme.Scheme
	if err := v1.AddToScheme(s); err != nil {
		glog.Fatalf("Error adding configmanagement resources to scheme: %v", err)
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		glog.Fatalf("Error adding configsync resources to scheme: %v", err)
	}

	// Use the DynamicRESTMapper as the default RESTMapper does not detect when
	// new types become available.
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		glog.Fatalf("Error creating DynamicRESTMapper: %v", err)
	}

	cl, err := client.New(cfg, client.Options{
		Scheme: s,
		Mapper: mapper,
	})
	if err != nil {
		glog.Fatalf("failed to create client: %v", err)
	}

	// Configure the Applier.
	genericClient := syncerclient.New(cl, metrics.APICallDuration)
	baseApplier, err := reconcile.NewApplierForMultiRepo(cfg, genericClient)
	if err != nil {
		glog.Fatalf("Instantiating Applier: %v", err)
	}

	var a *applier.Applier
	if opts.ReconcilerScope == declared.RootReconciler {
		a = applier.NewRootApplier(cl)
	} else {
		a = applier.NewNamespaceApplier(cl, opts.ReconcilerScope)
	}

	// Configure the Remediator.
	decls := &declared.Resources{}

	// Get a separate config for the remediator to talk to the apiserver since
	// we want a longer REST config timeout for the remediator to avoid restarting
	// idle watches too frequently.
	cfgForRemediator, err := restconfig.NewRestConfig(watch.RESTConfigTimeout)
	if err != nil {
		glog.Fatalf("failed to create rest config for the remediator: %v", err)
	}

	rem, err := remediator.New(opts.ReconcilerScope, cfgForRemediator, baseApplier, decls, opts.NumWorkers)
	if err != nil {
		glog.Fatalf("Instantiating Remediator: %v", err)
	}

	// Configure the Parser.
	var parser parse.Parser
	fs := parse.FileSource{
		GitDir:       opts.GitRoot,
		RepoRoot:     opts.RepoRoot,
		HydratedRoot: opts.HydratedRoot,
		HydratedLink: opts.HydratedLink,
		PolicyDir:    opts.PolicyDir,
		GitRepo:      opts.GitRepo,
		GitBranch:    opts.GitBranch,
		GitRev:       opts.GitRev,
	}
	if opts.ReconcilerScope == declared.RootReconciler {
		parser, err = parse.NewRootRunner(opts.ClusterName, RootSyncName, opts.SourceFormat, &reader.File{}, cl,
			opts.FilesystemPollingFrequency, opts.ResyncPeriod, fs, opts.DiscoveryClient, decls, a, rem)
		if err != nil {
			glog.Fatalf("Instantiating Root Repository Parser: %v", err)
		}
	} else {
		parser, err = parse.NewNamespaceRunner(opts.ClusterName, RepoSyncName(string(opts.ReconcilerScope)), opts.ReconcilerScope, &reader.File{}, cl,
			opts.FilesystemPollingFrequency, opts.ResyncPeriod, fs, opts.DiscoveryClient, decls, a, rem)
		if err != nil {
			glog.Fatalf("Instantiating Namespace Repository Parser: %v", err)
		}
	}

	ctx := signals.SetupSignalHandler()

	// Start the Remediator (non-blocking).
	rem.Start(ctx)
	// Start the Parser (blocking).
	// This will not return until:
	// - the Context is cancelled, or
	// - its Done channel is closed.
	parse.Run(ctx, parser)
}
