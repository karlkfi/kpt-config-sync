package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	"github.com/google/nomos/pkg/reposync"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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

	var rs v1.RepoSync
	if err := r.client.Get(ctx, req.NamespacedName, &rs); err != nil {
		log.Info("unable to fetch RepoSync", "error", err)
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.validate(&rs); err != nil {
		log.Error(err, "RepoSync failed validation")
		reposync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, err
	}

	if err := r.validateNamespaceSecret(ctx, &rs); err != nil {
		log.Error(err, "RepoSync failed Secret validation required for installation")
		reposync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, nil
	}
	log.V(2).Info("secret found, proceeding with installation")

	// Create secret in config-management-system namespace using the
	// existing secret in the reposync.namespace.
	if err := secret.Put(ctx, &rs, r.client); err != nil {
		log.Error(err, "RepoSync failed secret creation")
		return controllerruntime.Result{}, nil
	}

	// Overwrite reconciler pod's configmaps.
	configMapDataHash, err := r.upsertConfigMap(ctx, &rs)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		reposync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod deployment.
	if err := r.upsertDeployment(ctx, &rs, configMapDataHash); err != nil {
		log.Error(err, "Failed to create/update Deployment")
		reposync.SetStalled(&rs, "Deployment", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}

	// Since there were no errors, we can clear any previous Stalled condition.
	reposync.ClearCondition(&rs, v1.RepoSyncStalled)
	err = r.updateStatus(ctx, &rs, log)
	return controllerruntime.Result{}, err
}

// SetupWithManager registers RepoSync controller with reconciler-manager.
func (r *RepoSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	// mapSecretToRepoSync define a mapping from the Secret object in the event to
	// the RepoSync object to reconcile.
	mapSecretToRepoSync := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      reposync.Name,
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1.RepoSync{}).
		// Watch Secrets and trigger Reconciles for RepoSync object.
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapSecretToRepoSync}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// TODO b/163405299 De-duplicate RepoSyncReconciler.upsertConfigMap() and RootSyncReconciler.upsertConfigMap().
func (r *RepoSyncReconciler) upsertConfigMap(ctx context.Context, rs *v1.RepoSync) ([]byte, error) {
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
		childCM.Name = buildRepoSyncName(rs.Namespace, cm)
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			data, err := mutateRepoSyncConfigMap(rs, &childCM)
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
func (r *RepoSyncReconciler) validate(rs *v1.RepoSync) error {
	if rs.Name != reposync.Name {
		// Please don't change the error message.
		return fmt.Errorf(
			"there must be at most one RepoSync resource declared per namespace. "+
				"'meta.name' must be 'repo-sync'. Instead found %q", rs.Name)
	}
	return nil
}

// validateNamespaceSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RepoSyncReconciler) validateNamespaceSecret(ctx context.Context, repoSync *v1.RepoSync) error {
	if strings.ToLower(repoSync.Spec.Auth) == gitSecretNone || strings.ToLower(repoSync.Spec.Auth) == gitSecretGCENode {
		// There is no Secret to check for the Config object.
		return nil
	}
	secret, err := validateSecretExist(ctx,
		repoSync.Spec.SecretRef.Name,
		repoSync.Namespace,
		r.client)
	if err != nil {
		return err
	}
	return validateSecretData(repoSync.Spec.Auth, secret)
}

func mutateRepoSyncConfigMap(rs *v1.RepoSync, cm *corev1.ConfigMap) (map[string]string, error) {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding ConfigMap is also deleted.
	cm.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	switch cm.Name {
	case buildRepoSyncName(rs.Namespace, reconciler):
		cm.Data = reconcilerData(rs.Namespace, rs.Spec.Dir)
	case buildRepoSyncName(rs.Namespace, SourceFormat):
		cm.Data = sourceFormatData(rs.Spec.SourceFormat)
	case buildRepoSyncName(rs.Namespace, gitSync):
		cm.Data = gitSyncData(rs.Spec.Git.Revision, rs.Spec.Git.Repo)
	default:
		return nil, errors.Errorf("unsupported ConfigMap: %q", cm.Name)
	}

	return cm.Data, nil
}

func (r *RepoSyncReconciler) upsertDeployment(ctx context.Context, rs *v1.RepoSync, configMapDataHash []byte) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := nsParseDeployment(&childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}

	childDep.Name = buildRepoSyncName(rs.Namespace)
	childDep.Namespace = v1.NSConfigManagementSystem

	// CreateOrUpdate() first call Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.
	//
	// We make deep copy first so that we can set the declared fields as needed.
	declared := childDep.DeepCopyObject().(*appsv1.Deployment)
	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		return mutateRepoSyncDeployment(rs, &childDep, declared, configMapDataHash)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		rs.Status.Reconciler = childDep.Name
		msg := fmt.Sprintf("Reconciler deployment was %s", op)
		reposync.SetReconciling(rs, "Deployment", msg)
	}
	r.log.Info("Deployment successfully reconciled", executedOperation, op)
	return nil
}

func mutateRepoSyncDeployment(rs *v1.RepoSync, existing, declared *appsv1.Deployment, configMapDataHash []byte) error {
	// Update existing template.spec with reconciler template.spec.
	existing.Spec.Template.Spec = declared.Spec.Template.Spec

	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding Deployment is also deleted.
	existing.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	// Mutate Annotation with the hash of configmap.data from all the ConfigMap
	// reconciler creates/updates.
	core.SetAnnotation(&existing.Spec.Template, v1.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))

	templateSpec := &existing.Spec.Template.Spec

	var updatedVolumes []corev1.Volume
	// Mutate secret.secretname to secret reference specified in RepoSync CR.
	// Secret reference is the name of the secret used by git-sync container to
	// authenticate with the git repository using the authorization method specified
	// in the RepoSync CR.
	for _, volume := range templateSpec.Volumes {
		if volume.Name == gitCredentialVolume {
			volume.Secret.SecretName = secret.RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name)
		}
		updatedVolumes = append(updatedVolumes, volume)
	}
	templateSpec.Volumes = updatedVolumes

	var updatedContainers []corev1.Container
	// Mutate spec.Containers to update name, configmap references and volumemounts.
	for _, container := range templateSpec.Containers {
		switch container.Name {
		case reconciler:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRepoSyncName(rs.Namespace, reconciler)] = pointer.BoolPtr(false)
			configmapRef[buildRepoSyncName(rs.Namespace, SourceFormat)] = pointer.BoolPtr(true)
			container.EnvFrom = envFromSources(configmapRef)
		case gitSync:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRepoSyncName(rs.Namespace, gitSync)] = pointer.BoolPtr(false)
			container.EnvFrom = envFromSources(configmapRef)
		default:
			return errors.Errorf("unknown container in reconciler deployment template: %q", container.Name)
		}
		updatedContainers = append(updatedContainers, container)
	}
	templateSpec.Containers = updatedContainers
	return nil
}

func (r *RepoSyncReconciler) updateStatus(ctx context.Context, rs *v1.RepoSync, log logr.Logger) error {
	rs.Status.ObservedGeneration = rs.Generation
	err := r.client.Status().Update(ctx, rs)
	if err != nil {
		log.Error(err, "failed to update RepoSync status")
	}
	return err
}
