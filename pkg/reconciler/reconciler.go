package reconciler

import (
	"context"
	"time"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/rootsync"
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
	//
	// At most one Reconciler may have a given value for Scope on a cluster. More
	// than one results in undefined behavior.
	ReconcilerScope declared.Scope
	// ApplierResyncPeriod is the period of time between forced re-sync runs of
	// the Applier. At the end of each period, the Applier will re-apply its
	// current set of declared resources to the cluster.
	ApplierResyncPeriod time.Duration
	// GitPollingFrequency is how often to check the local git repository for
	// changes.
	GitPollingFrequency time.Duration
	// GitRoot is the absolute path to the Git repository.
	// Usually contains a symlink that must be resolved every time before parsing.
	GitRoot cmpath.Absolute
	// GitRef is the git revision being synced.
	GitRef string
	// GitRepo is the git repo being synced.
	GitRepo string
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
	var parser parse.Runnable
	if opts.ReconcilerScope == declared.RootReconciler {
		parser, err = parse.NewRootParser(opts.ClusterName, opts.SourceFormat, &filesystem.FileReader{}, mgr.GetClient(),
			opts.GitPollingFrequency, opts.GitRoot, opts.PolicyDir, opts.GitRef, opts.GitRepo, opts.DiscoveryInterfaceGetter)
		if err != nil {
			glog.Fatalf("Instantiating Root Repository Parser: %v", err)
		}
	} else {
		parser = parse.NewNamespaceParser(opts.ReconcilerScope, &filesystem.FileReader{}, mgr.GetClient(),
			opts.GitPollingFrequency, opts.GitRoot, opts.PolicyDir, opts.GitRef, opts.GitRepo, opts.DiscoveryInterfaceGetter)
	}

	// Right before we start everything, mark the RootSync or RepoSync as no longer
	// Reconciling.
	if opts.ReconcilerScope == declared.RootReconciler {
		removeRootSyncCondition(ctx, mgr.GetClient(), v1.RootSyncReconciling)
	} else {
		removeRepoSyncCondition(ctx, mgr.GetClient(), opts.ReconcilerScope, v1.RepoSyncReconciling)
	}

	// Start the Remediator.
	stopChan := signals.SetupSignalHandler()
	rem.Start(stoppableContext(ctx, stopChan))
	// Start the Applier.
	a.Run(ctx, opts.ApplierResyncPeriod, stopChan)
	// Start the Parser.
	// This will not return until:
	// - the Context is cancelled, or
	// - its Done channel is closed.
	parser.Run(stoppableContext(ctx, stopChan))
}

// removeRepoSyncCondition loops (with exponential backoff) until it is able to
// remove a status Condition from the reconciler's RepoSync.
func removeRepoSyncCondition(ctx context.Context, cl client.Client, namespace declared.Scope, cond v1.RepoSyncConditionType) {
	childCtx, cancel := context.WithCancel(ctx)
	wait.UntilWithContext(childCtx, func(childCtx context.Context) {
		var rs v1.RepoSync
		if err := cl.Get(childCtx, reposync.ObjectKey(namespace), &rs); err != nil {
			glog.Errorf("Failed to get RepoSync for %s reconciler: %v", namespace, err)
			return
		}

		reposync.ClearCondition(&rs, cond)
		if err := cl.Status().Update(childCtx, &rs); err != nil {
			glog.Errorf("Failed to update RepoSync status from %s reconciler: %v", namespace, err)
		} else {
			cancel()
		}
	}, time.Second)
}

// removeRootSyncCondition loops (with exponential backoff) until it is able to
// remove a status Condition from the reconciler's RootSync.
func removeRootSyncCondition(ctx context.Context, cl client.Client, cond v1.RootSyncConditionType) {
	childCtx, cancel := context.WithCancel(ctx)
	wait.UntilWithContext(childCtx, func(childCtx context.Context) {
		var rs v1.RootSync
		if err := cl.Get(childCtx, rootsync.ObjectKey(), &rs); err != nil {
			glog.Errorf("Failed to get RootSync: %v", err)
			return
		}

		rootsync.ClearCondition(&rs, cond)
		if err := cl.Status().Update(childCtx, &rs); err != nil {
			glog.Errorf("Failed to update RootSync status: %v", err)
		} else {
			cancel()
		}
	}, time.Second)
}

// stoppableContext returns a Context that will be canceled when the given stop
// channel is closed.
func stoppableContext(ctx context.Context, stopChannel <-chan struct{}) context.Context {
	stoppable, cancel := context.WithCancel(ctx)
	go func() {
		<-stopChannel
		cancel()
	}()
	return stoppable
}
