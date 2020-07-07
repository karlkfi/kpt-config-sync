package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

// Reconcile the RootSync resource.
func (r *RootSyncReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := r.log.WithValues("rootsync", req.NamespacedName)

	var rootSync v1.RootSync
	if err := r.client.Get(ctx, req.NamespacedName, &rootSync); err != nil {
		log.Info("unable to fetch RootSync", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var op controllerutil.OperationResult
	var err error

	if op, err = r.upsertConfigMap(ctx, req, rootSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}
	log.Info("ConfigMap successfully reconciled", executedOperation, op)

	if op, err = r.upsertDeployment(ctx, req, rootSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}
	log.Info("Deployment successfully reconciled", executedOperation, op)

	return ctrl.Result{}, nil
}

// SetupWithManager registers RootSync controller with reconciler-manager.
func (r *RootSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.RootSync{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RootSyncReconciler) upsertConfigMap(ctx context.Context, req ctrl.Request, rootSync v1.RootSync) (controllerutil.OperationResult, error) {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.
	var childCM corev1.ConfigMap
	childCM.Name = rootSyncReconcilerName
	childCM.Namespace = v1.NSConfigManagementSystem
	op, err := ctrl.CreateOrUpdate(ctx, r.client, &childCM, func() error {
		mutateRootSyncConfigMap(rootSync, &childCM)
		return nil
	})
	if err != nil {
		return "", err
	}
	return op, nil
}

func mutateRootSyncConfigMap(rs v1.RootSync, cm *corev1.ConfigMap) {
	// OwnerReferences, so that when the RootSync CustomResource is deleted,
	// the corresponding ConfigMap is also deleted.
	cm.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	cm.Data = configMapData(rs.Spec.Revision, rs.Spec.Repo)
}

func (r *RootSyncReconciler) upsertDeployment(ctx context.Context, req ctrl.Request, rootSync v1.RootSync) (controllerutil.OperationResult, error) {
	var childDep appsv1.Deployment
	if err := parseDeployment(&childDep); err != nil {
		return "", errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = rootSyncReconcilerName
	childDep.Namespace = v1.NSConfigManagementSystem
	op, err := ctrl.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		mutateRootSyncDeployment(rootSync, &childDep)
		return nil
	})
	if err != nil {
		return "", err
	}
	r.log.Info("Config for the deployment", "Environment Variable", childDep.Spec.Template.Spec.Containers[0].EnvFrom)
	return op, nil
}

func mutateRootSyncDeployment(rs v1.RootSync, de *appsv1.Deployment) {
	// OwnerReferences, so that when the RootSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	de.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	templateSpec := &de.Spec.Template.Spec
	// TODO Update upon addition of additional containers.
	container := &templateSpec.Containers[0]
	container.EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: rootSyncReconcilerName,
				},
			},
		},
	}
}
