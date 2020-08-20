package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
func (r *RootSyncReconciler) Reconcile(req controllerruntime.Request) (controllerruntime.Result, error) {
	ctx := context.TODO()
	log := r.log.WithValues("rootsync", req.NamespacedName)

	var rootSync v1.RootSync
	if err := r.client.Get(ctx, req.NamespacedName, &rootSync); err != nil {
		log.Info("unable to fetch RootSync", "error", err)
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.validate(rootSync); err != nil {
		log.Error(err, "failed to validate RootSync request")
		return controllerruntime.Result{}, nil
	}

	if err := r.validateRootSecret(ctx, &rootSync); err != nil {
		log.Error(err, "RootSync failed Secret validation required for installation")
		return controllerruntime.Result{}, nil
	}
	log.V(2).Info("secret found, proceeding with installation")

	// Overwrite git-importer pod's configmaps.
	configMapDataHash, err := r.upsertConfigMap(ctx, rootSync)
	if err != nil {
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite git-importer pod deployment.
	if err := r.upsertDeployment(ctx, rootSync, configMapDataHash); err != nil {
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}

	return controllerruntime.Result{}, nil
}

// SetupWithManager registers RootSync controller with reconciler-manager.
func (r *RootSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	// mapSecretToRootSync define a mapping from the Secret object in the event to
	// the RootSync object to reconcile.
	mapSecretToRootSync := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      rootSyncName,
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1.RootSync{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapSecretToRootSync}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RootSyncReconciler) upsertConfigMap(ctx context.Context, rootSync v1.RootSync) ([]byte, error) {
	// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
	// the object must be performed.
	// The name and namespace  must be filled in prior to calling CreateOrUpdate()
	//
	// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.

	// configMapData contain ConfigMap data.
	configMapData := make(map[string]map[string]string)

	// CreateOrUpdate configmaps for Root Reconciler.
	for _, cm := range reconcilerConfigMaps {
		var childCM corev1.ConfigMap
		childCM.Name = buildRootSyncName(cm)
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			data, err := mutateRootSyncConfigMap(rootSync, &childCM)
			configMapData[childCM.Name] = data
			return err
		})
		if err != nil {
			return nil, err
		}
		r.log.Info("ConfigMap successfully reconciled", executedOperation, op)
	}

	// hash return 128 bit FNV-1 hash of data of all the configmaps created by the controller.
	// Reconciler deployment's template.spec annotated with the hash. Deployment is recreated in the
	// event of configmap update. More information go/csmr-update-deployment
	return hash(configMapData)
}

// validate guarantees the RootSync CR is correct. See go/config-sync-multi-repo-user-guide for
// details.
func (r *RootSyncReconciler) validate(rs v1.RootSync) error {
	if rs.Name != rootSyncName {
		// Please don't change the error message.
		return fmt.Errorf(
			"there must be exactly one RootSync resource declared. 'meta.name' must be "+
				"'root-sync'. Instead found %q", rs.Name)
	}
	return nil
}

// validateRootSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RootSyncReconciler) validateRootSecret(ctx context.Context, rootSync *v1.RootSync) error {
	if strings.ToLower(rootSync.Spec.Auth) == gitSecretNone || strings.ToLower(rootSync.Spec.Auth) == gitSecretGCENode {
		// There is no Secret to check for the Config object.
		return nil
	}
	secret, err := validateSecretExist(ctx,
		rootSync.Spec.SecretRef.Name,
		rootSync.Namespace,
		r.client)
	if err != nil {
		return err
	}
	return validateSecretData(rootSync.Spec.Auth, secret)
}

func mutateRootSyncConfigMap(rs v1.RootSync, cm *corev1.ConfigMap) (map[string]string, error) {
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
	case buildRootSyncName(SourceFormat):
		cm.Data = sourceFormatData(rs.Spec.SourceFormat)
	case buildRootSyncName(gitSync):
		cm.Data = gitSyncData(rs.Spec.Git.Revision, rs.Spec.Git.Repo)
	default:
		return nil, errors.Errorf("unsupported ConfigMap: %q", cm.Name)
	}
	return cm.Data, nil
}

func (r *RootSyncReconciler) upsertDeployment(ctx context.Context, rootSync v1.RootSync, configMapDataHash []byte) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := rsParseDeployment(&childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = buildRootSyncName()
	childDep.Namespace = v1.NSConfigManagementSystem

	// CreateOrUpdate() first call Get() on the object. If the
	// object does not exist, Create() will be called. If it does exist, Update()
	// will be called. Just before calling either Create() or Update(), the mutate
	// callback will be called.
	//
	// We make deep copy first so that we can set the declared fields as needed.
	declared := childDep.DeepCopyObject().(*appsv1.Deployment)
	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childDep, func() error {
		return mutateRootSyncDeployment(rootSync, &childDep, declared, configMapDataHash)
	})
	if err != nil {
		return err
	}
	r.log.Info("Deployment successfully reconciled", executedOperation, op)
	return nil
}

func mutateRootSyncDeployment(rs v1.RootSync, existing, declared *appsv1.Deployment, configMapDataHash []byte) error {
	// Update existing template.spec with reconciler template.spec.
	existing.Spec.Template.Spec = declared.Spec.Template.Spec

	// OwnerReferences, so that when the RootSync CustomResource is deleted,
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
			volume.Secret.SecretName = rs.Spec.SecretRef.Name
		}
		updatedVolumes = append(updatedVolumes, volume)
	}
	templateSpec.Volumes = updatedVolumes

	var updatedContainers []corev1.Container
	// Mutate spec.Containers to update configmap references.
	//
	// ConfigMap references are updated for the respective containers.
	for _, container := range templateSpec.Containers {
		switch container.Name {
		case importer:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRootSyncName(importer)] = pointer.BoolPtr(false)
			configmapRef[buildRootSyncName(SourceFormat)] = pointer.BoolPtr(true)
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
