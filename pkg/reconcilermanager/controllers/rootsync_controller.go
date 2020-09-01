package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	"github.com/google/nomos/pkg/rootsync"
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

// RootSyncReconciler reconciles a RootSync object
type RootSyncReconciler struct {
	reconcilerBase
	clusterName string
}

// NewRootSyncReconciler returns a new RootSyncReconciler.
func NewRootSyncReconciler(cn string, c client.Client, l logr.Logger, s *runtime.Scheme) *RootSyncReconciler {
	return &RootSyncReconciler{
		reconcilerBase: reconcilerBase{
			client: c,
			log:    l,
			scheme: s,
		},
		clusterName: cn,
	}
}

// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=rootsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configmanagement.gke.io,resources=rootsyncs/status,verbs=get;update;patch

// Reconcile the RootSync resource.
func (r *RootSyncReconciler) Reconcile(req controllerruntime.Request) (controllerruntime.Result, error) {
	ctx := context.TODO()
	log := r.log.WithValues("rootsync", req.NamespacedName)

	var rs v1alpha1.RootSync
	if err := r.client.Get(ctx, req.NamespacedName, &rs); err != nil {
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.validate(&rs); err != nil {
		log.Error(err, "RootSync failed validation")
		rootsync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, err
	}

	if err := r.validateRootSecret(ctx, &rs); err != nil {
		log.Error(err, "RootSync failed Secret validation required for installation")
		rootsync.SetStalled(&rs, "Secret", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, err
	}
	log.V(2).Info("secret found, proceeding with installation")

	// Overwrite reconciler pod's configmaps.
	configMapDataHash, err := r.upsertConfigMaps(ctx, rs)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		rootsync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod ServiceAccount.
	if err := r.upsertServiceAccount(ctx, &rs); err != nil {
		log.Error(err, "Failed to create/update Service Account")
		rootsync.SetStalled(&rs, "ServiceAccount", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ServiceAccount reconcile failed")
	}

	// Overwrite reconciler pod deployment.
	if err := r.upsertDeployment(ctx, &rs, configMapDataHash); err != nil {
		log.Error(err, "Failed to create/update Deployment")
		rootsync.SetStalled(&rs, "Deployment", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}

	// Since there were no errors, we can clear any previous Stalled condition.
	rootsync.ClearCondition(&rs, v1alpha1.RootSyncStalled)
	err = r.updateStatus(ctx, &rs, log)
	return controllerruntime.Result{}, err
}

// SetupWithManager registers RootSync controller with reconciler-manager.
func (r *RootSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	// mapSecretToRootSync define a mapping from the Secret object in the event to
	// the RootSync object to reconcile.
	mapSecretToRootSync := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			return []reconcile.Request{
				{NamespacedName: types.NamespacedName{
					Name:      rootsync.Name,
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1alpha1.RootSync{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapSecretToRootSync}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RootSyncReconciler) upsertConfigMaps(ctx context.Context, rs v1alpha1.RootSync) ([]byte, error) {
	ownRefs := ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	configMapData := make(map[string]map[string]string)

	mutations := []struct {
		cmName string
		data   map[string]string
	}{
		{
			cmName: buildRootSyncName(SourceFormat),
			data:   sourceFormatData(rs.Spec.SourceFormat),
		},
		{
			cmName: buildRootSyncName(gitSync),
			data:   gitSyncData(rs.Spec.Revision, rs.Spec.Branch, rs.Spec.Repo),
		},
		{
			cmName: buildRootSyncName(reconciler),
			data:   rootReconcilerData(declared.RootReconciler, rs.Spec.Dir, r.clusterName, rs.Spec.Repo, rs.Spec.Branch, rs.Spec.Revision),
		},
	}

	for _, mutation := range mutations {
		mut := func(cm *corev1.ConfigMap) error {
			cm.OwnerReferences = ownRefs
			cm.Data = mutation.data
			return nil
		}

		err := r.upsertConfigMap(ctx, mutation.cmName, mut)
		if err != nil {
			return nil, err
		}

		configMapData[mutation.cmName] = mutation.data
	}

	// hash return 128 bit FNV-1 hash of data of the configmaps created by the controller.
	// Reconciler deployment's spec.template.annotation updated with the hash to recreate the
	// deployment in the event of configmap update. More information: go/csmr-update-deployment.
	return hash(configMapData)
}

// validate guarantees the RootSync CR is correct. See go/config-sync-multi-repo-user-guide for
// details.
func (r *RootSyncReconciler) validate(rs *v1alpha1.RootSync) error {
	if rs.Name != rootsync.Name {
		// Please don't change the error message.
		return fmt.Errorf(
			"there must be exactly one RootSync resource declared. 'meta.name' must be "+
				"'root-sync'. Instead found %q", rs.Name)
	}
	return nil
}

// validateRootSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RootSyncReconciler) validateRootSecret(ctx context.Context, rootSync *v1alpha1.RootSync) error {
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

func (r *RootSyncReconciler) upsertServiceAccount(ctx context.Context, rs *v1alpha1.RootSync) error {
	var childSA corev1.ServiceAccount
	childSA.Name = buildRootSyncName()
	childSA.Namespace = v1.NSConfigManagementSystem

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childSA, func() error {
		return mutateRootSyncServiceAccount(rs, &childSA)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ServiceAccount successfully reconciled", executedOperation, op)
	}
	return nil
}

func mutateRootSyncServiceAccount(rs *v1alpha1.RootSync, sa *corev1.ServiceAccount) error {
	// OwnerReferences, so that when the RootSync CustomResource is deleted,
	// the corresponding ServiceAccount is also deleted.
	sa.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)
	return nil
}

func (r *RootSyncReconciler) upsertDeployment(ctx context.Context, rs *v1alpha1.RootSync, configMapDataHash []byte) error {
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
		return mutateRootSyncDeployment(rs, &childDep, declared, configMapDataHash)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("Deployment successfully reconciled", executedOperation, op)
		rs.Status.Reconciler = childDep.Name
		msg := fmt.Sprintf("Reconciler deployment was %s", op)
		rootsync.SetReconciling(rs, "Deployment", msg)
	}
	return nil
}

func mutateRootSyncDeployment(rs *v1alpha1.RootSync, existing, declared *appsv1.Deployment, configMapDataHash []byte) error {
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
	core.SetAnnotation(&existing.Spec.Template, v1alpha1.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))

	templateSpec := &existing.Spec.Template.Spec

	// Update ServiceAccountName. eg. root-reconciler.
	templateSpec.ServiceAccountName = buildRootSyncName()

	var updatedVolumes []corev1.Volume
	// Mutate secret.secretname to secret reference specified in RootSync CR.
	// Secret reference is the name of the secret used by git-sync container to
	// authenticate with the git repository using the authorization method specified
	// in the RootSync CR.
	for _, volume := range templateSpec.Volumes {
		if volume.Name == gitCredentialVolume {
			// Don't mount git-creds volume if auth is 'none' or 'gcenode'
			if secret.SkipForAuth(rs.Spec.Auth) {
				continue
			}
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
		case reconciler:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRootSyncName(reconciler)] = pointer.BoolPtr(false)
			configmapRef[buildRootSyncName(SourceFormat)] = pointer.BoolPtr(true)
			container.EnvFrom = envFromSources(configmapRef)
		case gitSync:
			configmapRef := make(map[string]*bool)
			configmapRef[buildRootSyncName(gitSync)] = pointer.BoolPtr(false)
			container.EnvFrom = envFromSources(configmapRef)
			// Don't mount git-creds volume if auth is 'none' or 'gcenode'.
			container.VolumeMounts = volumeMounts(rs.Spec.Auth,
				container.VolumeMounts)
		default:
			return errors.Errorf("unknown container in reconciler deployment template: %q", container.Name)
		}
		updatedContainers = append(updatedContainers, container)
	}
	templateSpec.Containers = updatedContainers

	return nil
}

func (r *RootSyncReconciler) updateStatus(ctx context.Context, rs *v1alpha1.RootSync, log logr.Logger) error {
	rs.Status.ObservedGeneration = rs.Generation
	err := r.client.Status().Update(ctx, rs)
	if err != nil {
		log.Error(err, "failed to update RootSync status")
	}
	return err
}
