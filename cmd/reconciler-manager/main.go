package main

import (
	"flag"
	"os"

	"github.com/go-logr/glogr"
	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/profiler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	clusterName = flag.String("cluster-name", os.Getenv(reconcilermanager.ClusterNameKey),
		"Cluster name to use for Cluster selection")

	reconcilerPollingPeriod = flag.Duration("reconciler-polling-period", controllers.PollingPeriod(reconcilermanager.ReconcilerPollingPeriod, configsync.DefaultReconcilerPollingPeriod),
		"How often the reconciler should poll the filesystem for updates to the source or rendered configs.")

	hydrationPollingPeriod = flag.Duration("hydration-polling-period", controllers.PollingPeriod(reconcilermanager.HydrationPollingPeriod, configsync.DefaultHydrationPollingPeriod),
		"How often the hydration-controller should poll the filesystem for rendering the DRY configs.")

	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// glogr flags
	_ = flag.Set("v", "1")
	_ = flag.Set("logtostderr", "true")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = configmanagementv1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	profiler.Service()
	ctrl.SetLogger(glogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	repoSync := controllers.NewRepoSyncReconciler(*clusterName, *reconcilerPollingPeriod, *hydrationPollingPeriod, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RepoSync"),
		mgr.GetScheme())
	if err := repoSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RepoSync")
		os.Exit(1)
	}

	rootSync := controllers.NewRootSyncReconciler(*clusterName, *reconcilerPollingPeriod, *hydrationPollingPeriod, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RootSync"),
		mgr.GetScheme())
	if err := rootSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RootSync")
		os.Exit(1)
	}

	otel := controllers.NewOtelReconciler(*clusterName, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("Otel"),
		mgr.GetScheme())
	if err := otel.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Otel")
		os.Exit(1)
	}

	// Register the OpenCensus views
	if err := metrics.RegisterReconcilerManagerMetricsViews(); err != nil {
		setupLog.Error(err, "failed to register OpenCensus views")
	}

	// Register the OC Agent exporter
	oce, err := metrics.RegisterOCAgentExporter()
	if err != nil {
		setupLog.Error(err, "failed to register the OC Agent exporter")
		os.Exit(1)
	}

	defer func() {
		if err := oce.Stop(); err != nil {
			setupLog.Error(err, "unable to stop the OC Agent exporter")
		}
	}()

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		// os.Exit(1) does not run deferred functions so explicitly stopping the OC Agent exporter.
		if err := oce.Stop(); err != nil {
			setupLog.Error(err, "unable to stop the OC Agent exporter")
		}
		os.Exit(1)
	}
}
