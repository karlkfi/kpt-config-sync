package main

import (
	"flag"
	"os"
	"time"

	"github.com/go-logr/glogr"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	clusterName = flag.String("cluster-name", os.Getenv(reconcilermanager.ClusterNameKey),
		"Cluster name to use for Cluster selection")

	filesystemPollingPeriod = flag.Duration("filesystem-polling-period", pollingPeriod(),
		"Period of time between checking the filesystem for udpates to the local Git repository.")

	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// glogr flags
	_ = flag.Set("v", "1")
	_ = flag.Set("logtostderr", "true")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = configmanagementv1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
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

	ctrl.SetLogger(glogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	repoSync := controllers.NewRepoSyncReconciler(*clusterName, *filesystemPollingPeriod, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RepoSync"),
		mgr.GetScheme())
	if err := repoSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RepoSync")
		os.Exit(1)
	}

	rootSync := controllers.NewRootSyncReconciler(*clusterName, *filesystemPollingPeriod, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RootSync"),
		mgr.GetScheme())
	if err := rootSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RootSync")
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
