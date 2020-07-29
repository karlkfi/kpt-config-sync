package controllers

import (
	"context"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Overwrite git-importer pod's configmaps.
	if err := r.upsertConfigMap(ctx, rootSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite git-importer pod deployment.
	if err := r.upsertDeployment(ctx, rootSync); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}

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

func (r *RootSyncReconciler) upsertConfigMap(ctx context.Context, rootSync v1.RootSync) error {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.

	// CreateOrUpdate configmaps for Root Reconciler.
	for _, cm := range reconcilerConfigMaps {
		var childCM corev1.ConfigMap
		childCM.Name = buildRootSyncName(cm)
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := ctrl.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			return mutateRootSyncConfigMap(rootSync, &childCM)
		})
		if err != nil {
			return err
		}
		// TODO(b/161892553) Restart deployment when a configmap is updated.
		r.log.Info("ConfigMap successfully reconciled", executedOperation, op)
	}
	return nil
}

func mutateRootSyncConfigMap(rs v1.RootSync, cm *corev1.ConfigMap) error {
	// OwnerReferences, so that when the RootSync CustomResource is deleted,
	// the corresponding ConfigMap is also deleted.
	cm.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	switch cm.Name {
	case buildRootSyncName(importer):
		cm.Data = importerData(rs.Spec.Git.Dir)
	case buildRootSyncName(sourceFormat):
		cm.Data = sourceFormatData(rs.Spec.SourceFormat)
	case buildRootSyncName(gitSync):
		cm.Data = gitSyncData(rs.Spec.Git.Revision, rs.Spec.Git.Repo)
	default:
		return errors.Errorf("unsupported ConfigMap: %q", cm.Name)
	}
	return nil
}

func (r *RootSyncReconciler) upsertDeployment(ctx context.Context, rootSync v1.RootSync) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := parseDeployment(deploymentConfig, &childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = buildRootSyncName()
	childDep.Namespace = v1.NSConfigManagementSystem
	op, err := ctrl.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		return mutateRootSyncDeployment(rootSync, &childDep)
	})
	if err != nil {
		return err
	}
	r.log.Info("Deployment successfully reconciled", executedOperation, op)
	return nil
}

func mutateRootSyncDeployment(rs v1.RootSync, de *appsv1.Deployment) error {
	// OwnerReferences, so that when the RootSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	de.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	templateSpec := &de.Spec.Template.Spec

	var updatedContainers []corev1.Container
	// Mutate spec.Containers to update configmap references.
	//
	// ConfigMap references are updated for the respective containers.
	for _, container := range templateSpec.Containers {
		switch container.Name {
		case importer:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRootSyncName(importer)] = pointer.BoolPtr(false)
			configmapRef[buildRootSyncName(sourceFormat)] = pointer.BoolPtr(true)
			container.EnvFrom = envFromSources(configmapRef)
		case gitSync:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRootSyncName(gitSync)] = pointer.BoolPtr(false)
			container.EnvFrom = envFromSources(configmapRef)
		case fsWatcher:
		default:
			return errors.Errorf("unsupported Container: %q", container.Name)
		}
		updatedContainers = append(updatedContainers, container)
	}
	templateSpec.Containers = updatedContainers
	return nil

}
