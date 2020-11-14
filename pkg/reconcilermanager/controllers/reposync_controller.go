package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	"github.com/google/nomos/pkg/reposync"
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

// RepoSyncReconciler reconciles a RepoSync object.
type RepoSyncReconciler struct {
	reconcilerBase
}

// NewRepoSyncReconciler returns a new RepoSyncReconciler.
func NewRepoSyncReconciler(clusterName string, pollingPeriod time.Duration, client client.Client, log logr.Logger, scheme *runtime.Scheme) *RepoSyncReconciler {
	return &RepoSyncReconciler{
		reconcilerBase: reconcilerBase{
			clusterName:             clusterName,
			client:                  client,
			log:                     log,
			scheme:                  scheme,
			filesystemPollingPeriod: pollingPeriod,
		},
	}
}

// +kubebuilder:rbac:groups=configsync.gke.io,resources=reposyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configsync.gke.io,resources=reposyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RepoSyncReconciler) Reconcile(req controllerruntime.Request) (controllerruntime.Result, error) {
	ctx := context.TODO()
	log := r.log.WithValues("reposync", req.NamespacedName)

	var rs v1alpha1.RepoSync
	if err := r.client.Get(ctx, req.NamespacedName, &rs); err != nil {
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}

	owRefs := ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	var err error
	if err = nonhierarchical.ValidateRepoSync(&rs); err != nil {
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
		log.Error(err, "RepoSync failed secret creation", "auth", rs.Spec.Auth)
		return controllerruntime.Result{}, nil
	}

	// Overwrite reconciler pod's configmaps.
	configMapDataHash, err := r.upsertConfigMaps(ctx, r.repoConfigMapMutations(&rs), owRefs)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		reposync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod ServiceAccount.
	if err := r.upsertServiceAccount(ctx, repoSyncName(rs.Namespace), owRefs); err != nil {
		log.Error(err, "Failed to create/update ServiceAccount")
		reposync.SetStalled(&rs, "ServiceAccount", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "ServiceAccount reconcile failed")
	}

	// Overwrite reconciler rolebinding.
	if err := r.upsertRoleBinding(ctx, &rs); err != nil {
		log.Error(err, "Failed to create/update RoleBinding")
		reposync.SetStalled(&rs, "RoleBinding", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "RoleBinding reconcile failed")
	}

	mut := r.mutationsFor(rs, configMapDataHash)

	// Upsert Namespace reconciler deployment.
	op, err := r.upsertDeployment(ctx, repoSyncName(rs.Namespace), v1.NSConfigManagementSystem, mut)
	if err != nil {
		log.Error(err, "Failed to create/update Deployment")
		reposync.SetStalled(&rs, "Deployment", err)
		_ = r.updateStatus(ctx, &rs, log)
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("Deployment successfully reconciled", executedOperation, op)
		rs.Status.Reconciler = repoSyncName(rs.Namespace)
		msg := fmt.Sprintf("Reconciler deployment was %s", op)
		reposync.SetReconciling(&rs, "Deployment", msg)
	}

	// Since there were no errors, we can clear any previous Stalled condition.
	reposync.ClearCondition(&rs, v1alpha1.RepoSyncStalled)
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
					Name:      v1alpha1.RepoSyncName,
					Namespace: a.Meta.GetNamespace(),
				}},
			}
		})

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1alpha1.RepoSync{}).
		// Watch Secrets and trigger Reconciles for RepoSync object.
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapSecretToRepoSync}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RepoSyncReconciler) repoConfigMapMutations(rs *v1alpha1.RepoSync) []configMapMutation {
	return []configMapMutation{
		{
			cmName: repoSyncResourceName(rs.Namespace, gitSync),
			data: gitSyncData(options{
				ref:        rs.Spec.Git.Revision,
				branch:     rs.Spec.Git.Branch,
				repo:       rs.Spec.Git.Repo,
				secretType: rs.Spec.Git.Auth,
				period:     v1alpha1.GetPeriodSecs(&rs.Spec.Git),
				proxy:      rs.Spec.Proxy,
			}),
		},
		{
			cmName: repoSyncResourceName(rs.Namespace, reconciler),
			data:   reconcilerData(r.clusterName, declared.Scope(rs.Namespace), &rs.Spec.Git, r.filesystemPollingPeriod.String()),
		},
	}
}

// validateNamespaceSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RepoSyncReconciler) validateNamespaceSecret(ctx context.Context, repoSync *v1alpha1.RepoSync) error {
	if secret.SkipForAuth(repoSync.Spec.Auth) {
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

func (r *RepoSyncReconciler) upsertRoleBinding(ctx context.Context, rs *v1alpha1.RepoSync) error {
	var childRB rbacv1.RoleBinding
	childRB.Name = repoSyncPermissionsName()
	childRB.Namespace = rs.Namespace

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childRB, func() error {
		return mutateRoleBinding(rs, &childRB)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("RoleBinding successfully reconciled", executedOperation, op)
	}
	return nil
}

func mutateRoleBinding(rs *v1alpha1.RepoSync, rb *rbacv1.RoleBinding) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding RoleBinding is also deleted.
	rb.OwnerReferences = ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	// Update rolereference.
	rb.RoleRef = rolereference(repoSyncPermissionsName(), "ClusterRole")

	var subjects []rbacv1.Subject
	subjects = append(subjects, subject(repoSyncName(rs.Namespace),
		configsync.ControllerNamespace,
		"ServiceAccount"))
	// Update subject.
	rb.Subjects = subjects

	return nil
}

func (r *RepoSyncReconciler) updateStatus(ctx context.Context, rs *v1alpha1.RepoSync, log logr.Logger) error {
	rs.Status.ObservedGeneration = rs.Generation
	err := r.client.Status().Update(ctx, rs)
	if err != nil {
		log.Error(err, "failed to update RepoSync status")
	}
	return err
}

func (r *RepoSyncReconciler) mutationsFor(rs v1alpha1.RepoSync, configMapDataHash []byte) mutateFn {
	return func(d *appsv1.Deployment) error {
		// OwnerReferences, so that when the RepoSync CustomResource is deleted,
		// the corresponding Deployment is also deleted.
		d.OwnerReferences = ownerReference(
			rs.GroupVersionKind().Kind,
			rs.Name,
			rs.UID,
		)
		// Mutate Annotation with the hash of configmap.data from all the ConfigMap
		// reconciler creates/updates.
		core.SetAnnotation(&d.Spec.Template, v1alpha1.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))
		templateSpec := &d.Spec.Template.Spec
		// Update ServiceAccountName. eg. ns-reconciler-<namespace>
		templateSpec.ServiceAccountName = repoSyncName(rs.Namespace)
		// Mutate secret.secretname to secret reference specified in RepoSync CR.
		// Secret reference is the name of the secret used by git-sync container to
		// authenticate with the git repository using the authorization method specified
		// in the RepoSync CR.
		secretName := secret.RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name)
		templateSpec.Volumes = filterVolumes(templateSpec.Volumes, rs.Spec.Auth, secretName)
		var updatedContainers []corev1.Container
		// Mutate spec.Containers to update name, configmap references and volumemounts.
		for _, container := range templateSpec.Containers {
			switch container.Name {
			case reconciler:
				configmapRef := make(map[string]*bool)
				configmapRef[repoSyncResourceName(rs.Namespace, reconciler)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
			case gitSync:
				configmapRef := make(map[string]*bool)
				configmapRef[repoSyncResourceName(rs.Namespace, gitSync)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				// Don't mount git-creds volume if auth is 'none' or 'gcenode'.
				container.VolumeMounts = volumeMounts(rs.Spec.Auth,
					container.VolumeMounts)
				// Update Environment variables for `token` Auth, which
				// passes the credentials as the Username and Password.
				if authTypeToken(rs.Spec.Auth) {
					container.Env = gitSyncTokenAuthEnv(secret.RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name))
				}
			default:
				return errors.Errorf("unknown container in reconciler deployment template: %q", container.Name)
			}
			updatedContainers = append(updatedContainers, container)
		}
		// Add container spec for the "gcenode-askpass-sidecar" (defined as
		// a constant) to the reconciler Deployment when the `Auth` is "gcenode".
		if authTypeGCENode(rs.Spec.Auth) {
			updatedContainers = append(updatedContainers, gceNodeAskPassSidecar())
		}
		templateSpec.Containers = updatedContainers
		return nil
	}
}
