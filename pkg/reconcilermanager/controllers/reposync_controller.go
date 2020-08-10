package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/nomos/pkg/core"
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
	ctx := context.TODO()
	log := r.log.WithValues("reposync", req.NamespacedName)

	var repoSync v1.RepoSync
	if err := r.client.Get(ctx, req.NamespacedName, &repoSync); err != nil {
		log.Info("unable to fetch RepoSync", "error", err)
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.validate(repoSync); err != nil {
		log.Error(err, "failed to validate RepoSync request")
		return controllerruntime.Result{}, nil
	}

	// Overwrite git-importer pod's configmaps.
	configMapDataHash, err := r.upsertConfigMap(ctx, repoSync)
	if err != nil {
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite git-importer pod deployment.
	if err := r.upsertDeployment(ctx, repoSync, configMapDataHash); err != nil {
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

// TODO b/163405299 De-duplicate RepoSyncReconciler.upsertConfigMap() and RootSyncReconciler.upsertConfigMap().
func (r *RepoSyncReconciler) upsertConfigMap(ctx context.Context, repoSync v1.RepoSync) ([]byte, error) {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.

	// configMapDataHash contain ConfigMap data.
	configMapData := make(map[string]map[string]string)

	// CreateOrUpdate configmaps for Namespace Reconciler.
	for _, cm := range reconcilerConfigMaps {
		var childCM corev1.ConfigMap
		childCM.Name = buildRepoSyncName(repoSync.Namespace, cm)
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			data, err := mutateRepoSyncConfigMap(repoSync, &childCM)
			configMapData[childCM.Name] = data
			return err
		})
		if err != nil {
			return nil, err
		}
		r.log.Info("ConfigMap successfully reconciled", executedOperation, op)
	}

	// hash return 128 bit FNV-1 hash of data of the configmaps created by the controller.
	// Reconciler deployment's spec.template.annotation updated with the hash to recreate the
	// deployment in the event of configmap update. More information: go/csmr-update-deployment.
	return hash(configMapData)
}

// validate guarantees the RootSync CR is correct. See go/config-sync-multi-repo-user-guide for
// details.
func (r *RepoSyncReconciler) validate(rs v1.RepoSync) error {
	if rs.Name != repoSyncName {
		// Please don't change the error message.
		return fmt.Errorf(
			"there must be at most one RepoSync resource declared per namespace. "+
				"'meta.name' must be 'repo-sync'. Instead found %q", rs.Name)
	}
	return nil
}

func mutateRepoSyncConfigMap(rs v1.RepoSync, cm *corev1.ConfigMap) (map[string]string, error) {
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
		return nil, errors.Errorf("unsupported ConfigMap: %q", cm.Name)
	}

	return cm.Data, nil
}

func (r *RepoSyncReconciler) upsertDeployment(ctx context.Context, repoSync v1.RepoSync, configMapDataHash []byte) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := nsParseDeployment(&childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = buildRepoSyncName(repoSync.Namespace)
	childDep.Namespace = v1.NSConfigManagementSystem
	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		return mutateRepoSyncDeployment(repoSync, &childDep, configMapDataHash)
	})
	if err != nil {
		return err
	}
	r.log.Info("Deployment successfully reconciled", executedOperation, op)
	return nil
}

func mutateRepoSyncDeployment(rs v1.RepoSync, de *appsv1.Deployment, configMapDataHash []byte) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	de.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	// Mutate Annotation with the hash of configmap.data from all the ConfigMap
	// reconciler creates/updates.
	core.SetAnnotation(&de.Spec.Template, v1.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))

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
