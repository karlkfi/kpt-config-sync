package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secrets"
	"github.com/google/nomos/pkg/rootsync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/raw/validate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ReconcilerType defines the type of a reconciler
type ReconcilerType string

const (
	// RootReconcilerType defines the type for a root reconciler
	RootReconcilerType = ReconcilerType("root")
	// NamespaceReconcilerType defines the type for a namespace reconciler
	NamespaceReconcilerType = ReconcilerType("namespace")
)

// RootSyncReconciler reconciles a RootSync object
type RootSyncReconciler struct {
	reconcilerBase
}

// NewRootSyncReconciler returns a new RootSyncReconciler.
func NewRootSyncReconciler(clusterName string, reconcilerPollingPeriod, hydrationPollingPeriod time.Duration, client client.Client, log logr.Logger, scheme *runtime.Scheme) *RootSyncReconciler {
	return &RootSyncReconciler{
		reconcilerBase: reconcilerBase{
			clusterName:             clusterName,
			client:                  client,
			log:                     log,
			scheme:                  scheme,
			reconcilerPollingPeriod: reconcilerPollingPeriod,
			hydrationPollingPeriod:  hydrationPollingPeriod,
		},
	}
}

// +kubebuilder:rbac:groups=configsync.gke.io,resources=rootsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configsync.gke.io,resources=rootsyncs/status,verbs=get;update;patch

// Reconcile the RootSync resource.
func (r *RootSyncReconciler) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	log := r.log.WithValues("rootsync", req.NamespacedName)
	start := time.Now()

	var rs v1beta1.RootSync
	if err := r.client.Get(ctx, req.NamespacedName, &rs); err != nil {
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		if apierrors.IsNotFound(err) {
			return controllerruntime.Result{}, nil
		}
		return controllerruntime.Result{}, status.APIServerError(err, "failed to get RootSync")
	}

	owRefs := ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)

	var err error
	if err = validate.GitSpec(rs.Spec.Git, &rs); err != nil {
		log.Error(err, "RootSync failed validation")
		rootsync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}

	if err := r.validateRootSecret(ctx, &rs); err != nil {
		log.Error(err, "RootSync failed Secret validation required for installation")
		rootsync.SetStalled(&rs, "Secret", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}
	log.V(2).Info("secret found, proceeding with installation")

	// Overwrite reconciler pod's configmaps.
	configMapDataHash, err := r.upsertConfigMaps(ctx, r.rootConfigMapMutations(ctx, &rs), owRefs)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		rootsync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod ServiceAccount.
	if err := r.upsertServiceAccount(ctx, reconciler.RootSyncName, rs.Spec.Git.Auth, rs.Spec.Git.GCPServiceAccountEmail, owRefs); err != nil {
		log.Error(err, "Failed to create/update Service Account")
		rootsync.SetStalled(&rs, "ServiceAccount", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "ServiceAccount reconcile failed")
	}

	// Overwrite reconciler clusterrolebinding.
	if err := r.upsertClusterRoleBinding(ctx, &rs); err != nil {
		log.Error(err, "Failed to create/update ClusterRoleBinding")
		rootsync.SetStalled(&rs, "ClusterRoleBinding", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "ClusterRoleBinding reconcile failed")
	}

	mut := r.mutationsFor(ctx, rs, configMapDataHash)

	// Upsert Root reconciler deployment.
	op, err := r.upsertDeployment(ctx, reconciler.RootSyncName, v1.NSConfigManagementSystem, mut)
	if err != nil {
		log.Error(err, "Failed to create/update Deployment")
		rootsync.SetStalled(&rs, "Deployment", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}
	if op == controllerutil.OperationResultNone {
		// check the reconciler deployment conditions.
		result, err := r.deploymentStatus(ctx, client.ObjectKey{
			Namespace: v1.NSConfigManagementSystem,
			Name:      reconciler.RootSyncName,
		})
		if err != nil {
			log.Error(err, "Failed to check reconciler deployment conditions")
			rootsync.SetStalled(&rs, "Deployment", err)
			_ = r.updateStatus(ctx, &rs, log)
			return controllerruntime.Result{}, err
		}

		// Update RepoSync status based on reconciler deployment condition result.
		switch result.status {
		case statusInProgress:
			// inProgressStatus indicates that the deployment is not yet
			// available. Hence update the Reconciling status condition.
			rootsync.SetReconciling(&rs, "Deployment", result.message)
			// Clear Stalled condition.
			rootsync.ClearCondition(&rs, v1beta1.RootSyncStalled)
		case statusFailed:
			// statusFailed indicates that the deployment failed to reconcile. Update
			// Reconciling status condition with appropriate message specifying the
			// reason ffor failure.
			rootsync.SetReconciling(&rs, "Deployment", result.message)
			// Set Stalled condition with the deployment statusFailed.
			rootsync.SetStalled(&rs, "Deployment", errors.New(string(result.status)))
		case statusCurrent:
			// currentStatus indicates that the deployment is available, which qualifies
			// to clear the Reconciling status condition in RepoSync.
			rootsync.ClearCondition(&rs, v1beta1.RootSyncReconciling)
			// Since there were no errors, we can clear any previous Stalled condition.
			rootsync.ClearCondition(&rs, v1beta1.RootSyncStalled)
		}
	} else {
		r.log.Info("Deployment successfully reconciled", operationSubjectName, reconciler.RootSyncName, executedOperation, op)
		rs.Status.Reconciler = reconciler.RootSyncName
		msg := fmt.Sprintf("Reconciler deployment was %s", op)
		rootsync.SetReconciling(&rs, "Deployment", msg)
	}

	err = r.updateStatus(ctx, &rs, log)
	metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
	return controllerruntime.Result{}, err
}

// SetupWithManager registers RootSync controller with reconciler-manager.
func (r *RootSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1beta1.RootSync{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(mapSecretToRootSync())).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RootSyncReconciler) rootConfigMapMutations(ctx context.Context, rs *v1beta1.RootSync) []configMapMutation {
	return []configMapMutation{
		{
			cmName: RootSyncResourceName(reconcilermanager.SourceFormat),
			data:   sourceFormatData(rs.Spec.SourceFormat),
		},
		{
			cmName: RootSyncResourceName(reconcilermanager.GitSync),
			data: gitSyncData(ctx, options{
				ref:         rs.Spec.Git.Revision,
				branch:      rs.Spec.Git.Branch,
				repo:        rs.Spec.Git.Repo,
				secretType:  rs.Spec.Git.Auth,
				period:      v1beta1.GetPeriodSecs(&rs.Spec.Git),
				proxy:       rs.Spec.Proxy,
				depth:       rs.Spec.Override.GitSyncDepth,
				noSSLVerify: rs.Spec.Git.NoSSLVerify,
			}),
		},
		{
			cmName: RootSyncResourceName(reconcilermanager.HydrationController),
			data:   hydrationData(&rs.Spec.Git, declared.RootReconciler, r.hydrationPollingPeriod.String()),
		},
		{
			cmName: RootSyncResourceName(reconcilermanager.Reconciler),
			data:   reconcilerData(r.clusterName, declared.RootReconciler, &rs.Spec.Git, r.reconcilerPollingPeriod.String()),
		},
	}
}

// validateRootSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RootSyncReconciler) validateRootSecret(ctx context.Context, rootSync *v1beta1.RootSync) error {
	if secrets.SkipForAuth(rootSync.Spec.Auth) {
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

func (r *RootSyncReconciler) upsertClusterRoleBinding(ctx context.Context, rs *v1beta1.RootSync) error {
	var childCRB rbacv1.ClusterRoleBinding
	childCRB.Name = rootSyncPermissionsName()

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCRB, func() error {
		return mutateRootSyncClusterRoleBinding(rs, &childCRB)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ClusterRoleBinding successfully reconciled", operationSubjectName, childCRB.Name, executedOperation, op)
	}
	return nil
}

func mutateRootSyncClusterRoleBinding(rs *v1beta1.RootSync, crb *rbacv1.ClusterRoleBinding) error {
	// OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the corresponding ClusterRoleBinding is also deleted.
	crb.OwnerReferences = []metav1.OwnerReference{
		ownerReference(
			rs.GroupVersionKind().Kind,
			rs.Name,
			rs.UID,
		),
	}

	// Update rolereference.
	crb.RoleRef = rolereference("cluster-admin", "ClusterRole")

	var subjects []rbacv1.Subject
	subjects = append(subjects, subject(reconciler.RootSyncName,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	// Update subject.
	crb.Subjects = subjects

	return nil
}

func (r *RootSyncReconciler) updateStatus(ctx context.Context, rs *v1beta1.RootSync, log logr.Logger) error {
	rs.Status.ObservedGeneration = rs.Generation
	err := r.client.Status().Update(ctx, rs)
	if err != nil {
		log.Error(err, "failed to update RootSync status")
	}
	return err
}

func (r *RootSyncReconciler) mutationsFor(ctx context.Context, rs v1beta1.RootSync, configMapDataHash []byte) mutateFn {
	return func(obj client.Object) error {
		d, ok := obj.(*appsv1.Deployment)
		if !ok {
			return errors.Errorf("expected appsv1 Deployment, got: %T", obj)
		}
		// OwnerReferences, so that when the RootSync CustomResource is deleted,
		// the corresponding Deployment is also deleted.
		d.OwnerReferences = []metav1.OwnerReference{
			ownerReference(
				rs.GroupVersionKind().Kind,
				rs.Name,
				rs.UID,
			),
		}

		// Mutate Annotation with the hash of configmap.data from all the ConfigMap
		// reconciler creates/updates.
		core.SetAnnotation(&d.Spec.Template, metadata.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))

		// Add unique reconciler label
		core.SetLabel(&d.Spec.Template, metadata.ReconcilerLabel, reconciler.RootSyncName)

		templateSpec := &d.Spec.Template.Spec

		// Update ServiceAccountName.
		templateSpec.ServiceAccountName = reconciler.RootSyncName
		// The Deployment object fetched from the API server has the field defined.
		// Update DeprecatedServiceAccount to avoid discrepancy in equality check.
		templateSpec.DeprecatedServiceAccount = reconciler.RootSyncName

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
			case reconcilermanager.Reconciler:
				configmapRef := make(map[string]*bool)
				configmapRef[RootSyncResourceName(reconcilermanager.Reconciler)] = pointer.BoolPtr(false)
				configmapRef[RootSyncResourceName(reconcilermanager.SourceFormat)] = pointer.BoolPtr(true)
				container.EnvFrom = envFromSources(configmapRef)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(RootReconcilerType))
			case reconcilermanager.HydrationController:
				configmapRef := make(map[string]*bool)
				configmapRef[RootSyncResourceName(reconcilermanager.HydrationController)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(RootReconcilerType))
			case reconcilermanager.GitSync:
				configmapRef := make(map[string]*bool)
				configmapRef[RootSyncResourceName(reconcilermanager.GitSync)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				// Don't mount git-creds volume if auth is 'none' or 'gcenode'.
				container.VolumeMounts = volumeMounts(rs.Spec.Auth,
					container.VolumeMounts)
				// Update Environment variables for `token` Auth, which
				// passes the credentials as the Username and Password.
				secretName := rs.Spec.SecretRef.Name
				if authTypeToken(rs.Spec.Auth) {
					container.Env = gitSyncTokenAuthEnv(secretName)
				}
				keys := secrets.GetKeys(ctx, r.client, rs.Spec.SecretRef.Name, rs.Namespace)
				container.Env = append(container.Env, gitSyncHTTPSProxyEnv(secretName, keys)...)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(RootReconcilerType))
			case metrics.OtelAgentName:
				// The no-op case to avoid unknown container error after
				// first-ever reconcile.
			case GceNodeAskpassSidecarName:
				// container gcenode-askpass-sidecar is added to the reconciler
				// deployment when auth: gcenode or auth: gcpserveraccount.
				configureGceNodeAskPass(&container)
			default:
				return errors.Errorf("unknown container in reconciler deployment template: %q", container.Name)
			}
			updatedContainers = append(updatedContainers, container)
		}

		// Add container spec for the "gcenode-askpass-sidecar" (defined as
		// a constant) to the reconciler Deployment when the `Auth` is "gcenode".
		// The container is added first time when the reconciler deployment is created.
		switch rs.Spec.Auth {
		case configsync.GitSecretGCPServiceAccount, configsync.GitSecretGCENode:
			if !containsGCENodeAskPassSidecar(updatedContainers) {
				sidecar := gceNodeAskPassSidecar()
				updatedContainers = append(updatedContainers, sidecar)
			}
		}
		templateSpec.Containers = updatedContainers
		return nil
	}
}
