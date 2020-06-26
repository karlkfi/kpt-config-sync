package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

// RootSyncReconciler reconciles a RootSync object
type RootSyncReconciler struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewRootSyncReconciler returns a new RootSyncReconciler.
func NewRootSyncReconciler(c client.Client, l logr.Logger, s *runtime.Scheme) *RootSyncReconciler {
	return &RootSyncReconciler{
		client: c,
		log:    l,
		scheme: s,
	}
}

// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=rootsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=rootsyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RootSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.TODO()
	_ = r.log.WithValues("rootsync", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager registers RootSync controller with reconciler-manager.
func (r *RootSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configmanagementv1.RootSync{}).
		Complete(r)
}
