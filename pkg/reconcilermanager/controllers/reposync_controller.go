package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// RepoSyncReconciler reconciles a RepoSync object.
type RepoSyncReconciler struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewRepoSyncReconciler returns a new RepoSyncReconciler.
func NewRepoSyncReconciler(c client.Client, l logr.Logger, s *runtime.Scheme) *RepoSyncReconciler {
	return &RepoSyncReconciler{
		client: c,
		log:    l,
		scheme: s,
	}
}

// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=reposyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RepoSyncReconciler) Reconcile(req controllerruntime.Request) (controllerruntime.Result, error) {
	// TODO b/160179150 Pass context from the binary where the controllers are registered.
	ctx := context.TODO()
	log := r.log.WithValues("reposync", req.NamespacedName)

	var repoSync v1.RepoSync
	if err := r.client.Get(ctx, req.NamespacedName, &repoSync); err != nil {
		log.Info("unable to fetch RepoSync", "error", err)
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	// Overwrite git-importer pod's configmaps.
	if err := r.upsertConfigMap(ctx, repoSync); err != nil {
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite git-importer pod deployment.
	if err := r.upsertDeployment(ctx, repoSync); err != nil {
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}

	return controllerruntime.Result{}, nil
}

// SetupWithManager registers RepoSync controller with reconciler-manager.
func (r *RepoSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1.RepoSync{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RepoSyncReconciler) upsertConfigMap(ctx context.Context, repoSync v1.RepoSync) error {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.

	// CreateOrUpdate configmaps for Namespace Reconciler.
	for _, cm := range reconcilerConfigMaps {
		var childCM corev1.ConfigMap
		childCM.Name = buildRepoSyncName(repoSync.Namespace, cm)
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			return mutateRepoSyncConfigMap(repoSync, &childCM)
		})
		if err != nil {
			return err
		}
		// TODO(b/161892553) Restart deployment when a configmap is updated.
		r.log.Info("ConfigMap successfully reconciled", executedOperation, op)
	}
	return nil
}

func mutateRepoSyncConfigMap(rs v1.RepoSync, cm *corev1.ConfigMap) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding ConfigMap is also deleted.
	cm.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	switch cm.Name {
	case buildRepoSyncName(rs.Namespace, importer):
		cm.Data = importerData(rs.Spec.Git.Dir)
	case buildRepoSyncName(rs.Namespace, SourceFormat):
		cm.Data = sourceFormatData(rs.Spec.SourceFormat)
	case buildRepoSyncName(rs.Namespace, gitSync):
		cm.Data = gitSyncData(rs.Spec.Git.Revision, rs.Spec.Git.Repo)
	default:
		return errors.Errorf("unsupported ConfigMap: %q", cm.Name)
	}
	return nil
}

func (r *RepoSyncReconciler) upsertDeployment(ctx context.Context, repoSync v1.RepoSync) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := parseDeployment(&childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = buildRepoSyncName(repoSync.Namespace)
	childDep.Namespace = v1.NSConfigManagementSystem
	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		return mutateRepoSyncDeployment(repoSync, &childDep)
	})
	if err != nil {
		return err
	}
	r.log.Info("Deployment successfully reconciled", executedOperation, op)
	return nil
}

func mutateRepoSyncDeployment(rs v1.RepoSync, de *appsv1.Deployment) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	de.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	templateSpec := &de.Spec.Template.Spec

	var updatedContainers []corev1.Container
	// Mutate spec.Containers to update name and configmap references. The names
	// of containers are updated in the format repoSyncReconcilerPrefix + namespace +
	// containerName e.g. ns-reconciler-bookinfo-importer.
	//
	// In addition to updating the names for each Namespace Reconciler, configmap
	// references are updated corresponsing to the Namespace Reconciler.
	for _, container := range templateSpec.Containers {
		switch container.Name {
		case importer:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRepoSyncName(rs.Namespace, importer)] = pointer.BoolPtr(false)
			configmapRef[buildRepoSyncName(rs.Namespace, SourceFormat)] = pointer.BoolPtr(true)
			container.EnvFrom = envFromSources(configmapRef)
		case gitSync:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRepoSyncName(rs.Namespace, gitSync)] = pointer.BoolPtr(false)
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
