package reconciler

import (
	"context"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/configsync"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/reposync"
	syncerclient "github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Options contains the settings for a reconciler process.
type Options struct {
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
	ReconcilerScope string
	// ApplierResyncPeriod is the period of time between forced re-sync runs of
	// the Applier. At the end of each period, the Applier will re-apply its
	// current set of declared resources to the cluster.
	ApplierResyncPeriod time.Duration
	// GitPollingFrequency is how often to check the local git repository for
	// changes.
	GitPollingFrequency time.Duration
	// GitRoot is the absolute path to the Git repository.
	// Usually contains a symlink that must be resolved every time before parsing.
	GitRoot string
	// PolicyDir is the relative path to the policies within the Git repository.
	PolicyDir cmpath.Relative
	// DiscoveryInterfaceGetter is how to fetch a new DiscoveryClient when the set
	// of available CRDs may have changed.
	// TODO(b/162958883): Determine if we can actually just used a
	//  CachedDiscoveryClient and just invalidate it each time, since that's
	//  simpler.
	DiscoveryInterfaceGetter discovery.ClientGetter
	// RootOptions is the set of options to fill in if this is configuring the
	// Root reconciler.
	// Unset for Namespace repositories.
	*RootOptions
}

// RootOptions are the options specific to parsing Root repositories.
type RootOptions struct {
	// ClusterName is the name of the cluster we are parsing configuration for.
	ClusterName string
	// SourceFormat is how the Root repository is structured.
	SourceFormat filesystem.SourceFormat
}

// Run configures and starts the various components of a reconciler process.
func Run(ctx context.Context, opts Options) {
	reconcile.SetFightThreshold(opts.FightDetectionThreshold)

	// Get a config to talk to the apiserver.
	cfg, err := restconfig.NewRestConfig()
	if err != nil {
		glog.Fatalf("failed to create rest config: %+v", err)
	}

	// TODO(b/159068994): Determine if we *actually* need a Manager.
	// Right now a lot of this is just cargoculted over.

	// Create a new Manager to provide shared dependencies and start components.
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		glog.Fatalf("Failed to create manager: %v", err)
	}

	// Set up Scheme for configmanagement resources.
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		glog.Fatalf("Error adding configmanagement resources to scheme: %v", err)
	}

	// Configure the Applier.
	genericClient := syncerclient.New(mgr.GetClient(), metrics.APICallDuration)
	baseApplier, err := reconcile.NewApplier(cfg, genericClient)
	if err != nil {
		glog.Fatalf("Instantiating Applier: %v", err)
	}

	var a *applier.Applier
	if opts.ReconcilerScope == declared.RootReconciler {
		a = applier.NewRootApplier(mgr.GetClient(), baseApplier)
	} else {
		a = applier.NewNamespaceApplier(mgr.GetClient(), baseApplier, opts.ReconcilerScope)
	}

	// Configure the Remediator.
	decls := declared.NewResources()

	rem, err := remediator.New(opts.ReconcilerScope, cfg, baseApplier, decls, opts.NumWorkers)
	if err != nil {
		glog.Fatalf("Instantiating Remediator: %v", err)
	}

	// Configure the Parser.
	gitRoot, err := cmpath.AbsoluteOS(opts.GitRoot)
	if err != nil {
		glog.Fatalf("Validating repository root path %q: %v", opts.GitRoot, err)
	}
	var parser parse.Runnable
	if opts.ReconcilerScope == declared.RootReconciler {
		parser, err = parse.NewRootParser(opts.ClusterName, opts.SourceFormat, &filesystem.FileReader{}, mgr.GetClient(),
			opts.GitPollingFrequency, gitRoot, opts.PolicyDir, opts.DiscoveryInterfaceGetter)
		if err != nil {
			glog.Fatalf("Instantiating Root Repository Parser: %v", err)
		}
	} else {
		parser = parse.NewNamespaceParser(opts.ReconcilerScope, &filesystem.FileReader{}, mgr.GetClient(),
			opts.GitPollingFrequency, gitRoot, opts.PolicyDir, opts.DiscoveryInterfaceGetter)
	}

	// Right before we start everything, mark the RepoSync as no longer
	// Reconciling.
	updateRepoSyncStatus(ctx, mgr.GetClient(), opts.ReconcilerScope)

	// Start the Remediator.
	stopChan := signals.SetupSignalHandler()
	rem.Start(configsync.StoppableContext(ctx, stopChan))
	// Start the Applier.
	a.Run(ctx, opts.ApplierResyncPeriod, stopChan)
	// Start the Parser.
	// This will not return until:
	// - the Context is cancelled, or
	// - its Done channel is closed.
	parser.Run(configsync.StoppableContext(ctx, stopChan))
}

// updateRepoSyncStatus loops (with exponential backoff) until it is able to
// remove the Reconciling status from the reconciler's RepoSync.
func updateRepoSyncStatus(ctx context.Context, cl client.Client, namespace string) {
	childCtx, cancel := context.WithCancel(ctx)
	wait.UntilWithContext(childCtx, func(childCtx context.Context) {
		var rs v1.RepoSync
		if err := cl.Get(childCtx, reposync.ObjectKey(namespace), &rs); err != nil {
			glog.Errorf("Failed to get RepoSync for %s reconciler: %v", namespace, err)
			return
		}

		reposync.ClearCondition(&rs, v1.RepoSyncReconciling)
		if err := cl.Status().Update(childCtx, &rs); err != nil {
			glog.Errorf("Failed to update RepoSync status from %s reconciler: %v", namespace, err)
		} else {
			cancel()
		}
	}, time.Second)
}
