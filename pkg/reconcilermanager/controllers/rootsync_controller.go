package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

	owRefs := ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

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
	configMapDataHash, err := r.upsertConfigMaps(ctx, r.rootConfigMapMutations(&rs), owRefs)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		rootsync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod ServiceAccount.
	if err := r.upsertServiceAccount(ctx, rootSyncReconcilerName, owRefs); err != nil {
		log.Error(err, "Failed to create/update Service Account")
		rootsync.SetStalled(&rs, "ServiceAccount", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ServiceAccount reconcile failed")
	}

	// Overwrite reconciler clusterrolebinding.
	if err := r.upsertClusterRoleBinding(ctx, &rs); err != nil {
		log.Error(err, "Failed to create/update ClusterRoleBinding")
		rootsync.SetStalled(&rs, "ClusterRoleBinding", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ClusterRoleBinding reconcile failed")
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
					Name:      v1alpha1.RootSyncName,
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

func (r *RootSyncReconciler) rootConfigMapMutations(rs *v1alpha1.RootSync) []configMapMutation {
	return []configMapMutation{
		{
			cmName: rootSyncResourceName(SourceFormat),
			data:   sourceFormatData(rs.Spec.SourceFormat),
		},
		{
			cmName: rootSyncResourceName(gitSync),
			data:   gitSyncData(rs.Spec.Revision, rs.Spec.Branch, rs.Spec.Repo, rs.Spec.Git.Auth, v1alpha1.GetPeriodSecs(&rs.Spec.Git)),
		},
		{
			cmName: rootSyncResourceName(reconciler),
			data:   rootReconcilerData(declared.RootReconciler, rs.Spec.Dir, r.clusterName, rs.Spec.Repo, rs.Spec.Branch, rs.Spec.Revision),
		},
	}
}

// validate guarantees the RootSync CR is correct. See go/config-sync-multi-repo-user-guide for
// details.
func (r *RootSyncReconciler) validate(rs *v1alpha1.RootSync) error {
	if rs.Name != v1alpha1.RootSyncName {
		// Please don't change the error message.
		return fmt.Errorf(
			"there must be exactly one RootSync resource declared. 'meta.name' must be "+
				"'root-sync'. Instead found %q", rs.Name)
	}
	return nil
}

// validateRootSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RootSyncReconciler) validateRootSecret(ctx context.Context, rootSync *v1alpha1.RootSync) error {
	if rootSync.Spec.Auth == v1alpha1.GitSecretNone || rootSync.Spec.Auth == v1alpha1.GitSecretGCENode {
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

func (r *RootSyncReconciler) upsertClusterRoleBinding(ctx context.Context, rs *v1alpha1.RootSync) error {
	var childCRB rbacv1.ClusterRoleBinding
	childCRB.Name = rootSyncPermissionsName()

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCRB, func() error {
		return mutateRootSyncClusterRoleBinding(rs, &childCRB)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ClusterRoleBinding successfully reconciled", executedOperation, op)
	}
	return nil
}

func mutateRootSyncClusterRoleBinding(rs *v1alpha1.RootSync, crb *rbacv1.ClusterRoleBinding) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding ClusterRoleBinding is also deleted.
	crb.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	// Update rolereference.
	crb.RoleRef = rolereference("cluster-admin", "ClusterRole")

	var subjects []rbacv1.Subject
	subjects = append(subjects, subject(rootSyncReconcilerName,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	// Update subject.
	crb.Subjects = subjects

	return nil
}

func (r *RootSyncReconciler) upsertDeployment(ctx context.Context, rs *v1alpha1.RootSync, configMapDataHash []byte) error {
	var childDep appsv1.Deployment
	// Parse the deployment.yaml mounted as configmap in Reconciler Managers deployment.
	if err := rsParseDeployment(&childDep); err != nil {
		return errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}
	childDep.Name = rootSyncReconcilerName
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

	// Update ServiceAccountName.
	templateSpec.ServiceAccountName = rootSyncReconcilerName

	// Mutate secret.secretname to secret reference specified in RootSync CR.
	// Secret reference is the name of the secret used by git-sync container to
	// authenticate with the git repository using the authorization method specified
	// in the RootSync CR.
	templateSpec.Volumes = filterVolumes(templateSpec.Volumes, rs.Spec.Auth, rs.Spec.SecretRef.Name)

	var updatedContainers []corev1.Container
	// Mutate spec.Containers to update configmap references.
	//
	// ConfigMap references are updated for the respective containers.
	for _, container := range templateSpec.Containers {
		switch container.Name {
		case reconciler:
			configmapRef := make(map[string]*bool)
			configmapRef[rootSyncResourceName(reconciler)] = pointer.BoolPtr(false)
			configmapRef[rootSyncResourceName(SourceFormat)] = pointer.BoolPtr(true)
			container.EnvFrom = envFromSources(configmapRef)
		case gitSync:
			configmapRef := make(map[string]*bool)
			configmapRef[rootSyncResourceName(gitSync)] = pointer.BoolPtr(false)
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
