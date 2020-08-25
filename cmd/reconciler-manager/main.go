package main

import (
	"flag"
	"os"

	"github.com/go-logr/glogr"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	// Root-Repo-only flag.
	clusterName = flag.String("cluster-name", os.Getenv("CLUSTER_NAME"),
		"Cluster name to use for Cluster selection")

	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// glogr flags
	_ = flag.Set("v", "1")
	_ = flag.Set("logtostderr", "true")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = configmanagementv1.AddToScheme(scheme)
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

	repoSync := controllers.NewRepoSyncReconciler(mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RepoSync"),
		mgr.GetScheme())
	if err := repoSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RepoSync")
		os.Exit(1)
	}

	rootSync := controllers.NewRootSyncReconciler(*clusterName, mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName("RootSync"),
		mgr.GetScheme())
	if err := rootSync.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RootSync")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
