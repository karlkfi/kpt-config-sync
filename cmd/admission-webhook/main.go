package main

import (
	"flag"
	"os"

	"github.com/go-logr/glogr"
	"github.com/google/nomos/pkg/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	setupLog = ctrl.Log.WithName("setup")

	restartOnSecretRefresh bool
)

func init() {
	ctrl.SetLogger(glogr.New())
}

func main() {
	// Default to true since not restarting doesn't make sense?
	flag.BoolVar(&restartOnSecretRefresh, "cert-restart-on-secret-refresh", true, "Kills the process when secrets are refreshed so that the pod can be restarted (secrets take up to 60s to be updated by running pods)")
	flag.Parse()

	setupLog.Info("starting manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		setupLog.Error(err, "starting manager")
		os.Exit(1)
	}

	setupLog.Info("creating certificate rotator for webhook")
	certDone, err := webhook.CreateCertsIfNeeded(mgr, true)
	if err != nil {
		setupLog.Error(err, "creating certificate rotator for webhook")
	}
	setupLog.Info("waiting for certificate rotator")
	<-certDone

	setupLog.Info("registering validating webhook")
	if err = webhook.AddValidator(mgr); err != nil {
		setupLog.Error(err, "registering validating webhook")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "running manager")
		os.Exit(1)
	}
}
