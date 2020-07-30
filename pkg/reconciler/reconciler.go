package reconciler

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/parse/declaredresources"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RootScope is a special constant for a reconciler which is running as the
// "root reconciler" (vs a namespace reconciler).
const RootScope = ":root"

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
}

// Run configures and starts the various components of a reconciler process.
func Run(opts Options) {
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

	// Configure Applier
	genericClient := client.New(mgr.GetClient(), metrics.APICallDuration)
	baseApplier, err := reconcile.NewApplier(mgr.GetConfig(), genericClient)

	if opts.ReconcilerScope == RootScope {
		applier.NewRootApplier(genericClient, baseApplier)
	} else {
		applier.NewNamespaceApplier(genericClient, baseApplier, opts.ReconcilerScope)
	}

	// Configure Remediator
	decls := declaredresources.NewDeclaredResources()
	if err != nil {
		glog.Fatalf("Instantiating Applier for Remediator: %v", err)
	}

	remediator.New(opts.ReconcilerScope, baseApplier, decls, opts.NumWorkers)

	// Configure Parser
	// TODO(b/162014057): configure the parser and get everything running.

	// Note that there should be a blocking call here.
}
