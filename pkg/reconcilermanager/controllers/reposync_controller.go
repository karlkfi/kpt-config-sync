package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configmanagementv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

// RepoSyncReconciler reconciles a RepoSync object
type RepoSyncReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RepoSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("reposync", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager registers RepoSync controller with reconciler-manager.
func (r *RepoSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configmanagementv1.RepoSync{}).
		Complete(r)
}
