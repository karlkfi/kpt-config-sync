// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"strings"
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
	"github.com/google/nomos/pkg/reposync"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/raw/validate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RepoSyncReconciler reconciles a RepoSync object.
type RepoSyncReconciler struct {
	reconcilerBase
	namespaces map[string]struct{}
}

// NewRepoSyncReconciler returns a new RepoSyncReconciler.
func NewRepoSyncReconciler(clusterName string, reconcilerPollingPeriod, hydrationPollingPeriod time.Duration, client client.Client, log logr.Logger, scheme *runtime.Scheme) *RepoSyncReconciler {
	return &RepoSyncReconciler{
		reconcilerBase: reconcilerBase{
			clusterName:             clusterName,
			client:                  client,
			log:                     log,
			scheme:                  scheme,
			reconcilerPollingPeriod: reconcilerPollingPeriod,
			hydrationPollingPeriod:  hydrationPollingPeriod,
		},
		namespaces: make(map[string]struct{}),
	}
}

// +kubebuilder:rbac:groups=configsync.gke.io,resources=reposyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=configsync.gke.io,resources=reposyncs/status,verbs=get;update;patch

// Reconcile the RepoSync resource.
func (r *RepoSyncReconciler) Reconcile(ctx context.Context, req controllerruntime.Request) (controllerruntime.Result, error) {
	log := r.log.WithValues("reposync", req.NamespacedName)
	start := time.Now()
	var err error
	var rs v1beta1.RepoSync

	if err = r.validateNamespaceName(req.NamespacedName.Namespace); err != nil {
		log.Error(err, "RepoSync namespace name failed validation")
		reposync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}

	if err := r.client.Get(ctx, req.NamespacedName, &rs); err != nil {
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		if apierrors.IsNotFound(err) {
			if _, ok := r.namespaces[req.Namespace]; !ok {
				log.Error(err, "The RepoSync reconciler does not manage a RepoSync object for the namespace", "namespace", req.Namespace)
				// return `controllerruntime.Result{}, nil` here to make sure the request will not be requeued.
				return controllerruntime.Result{}, nil
			}
			// Namespace controller resources are cleaned up when reposync no longer present.
			//
			// Note: Update cleanup resources in cleanupNSControllerResources(...) when
			// resources created by namespace controller changes.
			cleanupErr := r.cleanupNSControllerResources(ctx, req.Namespace, reconciler.NsReconcilerName(req.Namespace, req.Name))
			// if cleanupErr != nil, the request will be requeued.
			return controllerruntime.Result{}, cleanupErr
		}
		return controllerruntime.Result{}, status.APIServerError(err, "failed to get RepoSync")
	}

	r.namespaces[req.Namespace] = struct{}{}
	reconcilerName := reconciler.NsReconcilerName(rs.Namespace, rs.Name)

	if err = validate.GitSpec(rs.Spec.Git, &rs); err != nil {
		log.Error(err, "RepoSync failed validation")
		reposync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}

	repoCMMutations := r.repoConfigMapMutations(ctx, &rs, reconcilerName)
	if err = r.validateResourcesName(repoCMMutations); err != nil {
		log.Error(err, "Resource name failed validation")
		reposync.SetStalled(&rs, "Resource name validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}

	secretName := ReconcilerResourceName(reconcilerName, rs.Spec.SecretRef.Name)
	if errs := validation.IsDNS1123Label(secretName); err != nil {
		err = errors.New(strings.Join(errs, " "))
		log.Error(err, "Resource name failed validation")
		reposync.SetStalled(&rs, "Resource name validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		err = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, err
	}

	if err := r.validateNamespaceSecret(ctx, &rs); err != nil {
		log.Error(err, "RepoSync failed Secret validation required for installation")
		reposync.SetStalled(&rs, "Validation", err)
		// We intentionally overwrite the previous error here since we do not want
		// to return it to the controller runtime.
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, nil
	}

	reposyncLabelMap := map[string]string{
		metadata.SyncNamespaceLabel: rs.Namespace,
		metadata.SyncNameLabel:      rs.Name,
	}

	// Create secret in config-management-system namespace using the
	// existing secret in the reposync.namespace.
	if err := Put(ctx, &rs, r.client, reconcilerName, secretName); err != nil {
		log.Error(err, "RepoSync failed secret creation", "auth", rs.Spec.Auth)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, nil
	}

	// Overwrite reconciler pod's configmaps.
	configMapDataHash, err := r.upsertConfigMaps(ctx, repoCMMutations, reposyncLabelMap)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		reposync.SetStalled(&rs, "ConfigMap", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "ConfigMap reconcile failed")
	}

	// Overwrite reconciler pod ServiceAccount.
	if err := r.upsertServiceAccount(ctx, reconcilerName, rs.Spec.Git.Auth, rs.Spec.Git.GCPServiceAccountEmail, reposyncLabelMap); err != nil {
		log.Error(err, "Failed to create/update ServiceAccount")
		reposync.SetStalled(&rs, "ServiceAccount", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "ServiceAccount reconcile failed")
	}

	// Overwrite reconciler rolebinding.
	if err := r.upsertRoleBinding(ctx, rs.Namespace); err != nil {
		log.Error(err, "Failed to create/update RoleBinding")
		reposync.SetStalled(&rs, "RoleBinding", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "RoleBinding reconcile failed")
	}

	mut := r.mutationsFor(ctx, rs, configMapDataHash)

	// Upsert Namespace reconciler deployment.
	op, err := r.upsertDeployment(ctx, reconcilerName, v1.NSConfigManagementSystem, reposyncLabelMap, mut)
	if err != nil {
		log.Error(err, "Failed to create/update Deployment")
		reposync.SetStalled(&rs, "Deployment", err)
		_ = r.updateStatus(ctx, &rs, log)
		metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
		return controllerruntime.Result{}, errors.Wrap(err, "Deployment reconcile failed")
	}
	if op == controllerutil.OperationResultNone {
		// check the reconciler deployment conditions.
		result, err := r.deploymentStatus(ctx, client.ObjectKey{
			Namespace: v1.NSConfigManagementSystem,
			Name:      reconcilerName,
		})
		if err != nil {
			log.Error(err, "Failed to check reconciler deployment conditions")
			reposync.SetStalled(&rs, "Deployment", err)
			_ = r.updateStatus(ctx, &rs, log)
			return controllerruntime.Result{}, err
		}

		// Update RepoSync status based on reconciler deployment condition result.
		switch result.status {
		case statusInProgress:
			// inProgressStatus indicates that the deployment is not yet
			// available. Hence update the Reconciling status condition.
			reposync.SetReconciling(&rs, "Deployment", result.message)
			// Clear Stalled condition.
			reposync.ClearCondition(&rs, v1beta1.RepoSyncStalled)
		case statusFailed:
			// statusFailed indicates that the deployment failed to reconcile. Update
			// Reconciling status condition with appropriate message specifying the
			// reason for failure.
			reposync.SetReconciling(&rs, "Deployment", result.message)
			// Set Stalled condition with the deployment statusFailed.
			reposync.SetStalled(&rs, "Deployment", errors.New(string(result.status)))
		case statusCurrent:
			// currentStatus indicates that the deployment is available, which qualifies
			// to clear the Reconciling status condition in RepoSync.
			reposync.ClearCondition(&rs, v1beta1.RepoSyncReconciling)
			// Since there were no errors, we can clear any previous Stalled condition.
			reposync.ClearCondition(&rs, v1beta1.RepoSyncStalled)
		}
	} else {
		r.log.Info("Deployment successfully reconciled", operationSubjectName, reconcilerName, executedOperation, op)
		rs.Status.Reconciler = reconcilerName
		msg := fmt.Sprintf("Reconciler deployment was %s", op)
		reposync.SetReconciling(&rs, "Deployment", msg)
	}

	err = r.updateStatus(ctx, &rs, log)
	metrics.RecordReconcileDuration(ctx, metrics.StatusTagKey(err), start)
	return controllerruntime.Result{}, err
}

// SetupWithManager registers RepoSync controller with reconciler-manager.
func (r *RepoSyncReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	// Index the `gitSecretRefName` field, so that we will be able to lookup RepoSync be a referenced `SecretRef` name.
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1beta1.RepoSync{}, gitSecretRefField, func(rawObj client.Object) []string {
		rs := rawObj.(*v1beta1.RepoSync)
		if rs.Spec.Git.SecretRef.Name == "" {
			return nil
		}
		return []string{rs.Spec.Git.SecretRef.Name}
	}); err != nil {
		return err
	}

	return controllerruntime.NewControllerManagedBy(mgr).
		For(&v1beta1.RepoSync{}).
		// Custom Watch to trigger Reconcile for objects created by RepoSync controller.
		Watches(&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToRepoSyncs),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &appsv1.Deployment{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToRepoSync),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToRepoSync),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.ServiceAccount{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToRepoSync),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &rbacv1.RoleBinding{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectToRepoSync),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

// mapSecretToRepoSyncs define a mapping from the Secret object to its attached
// RepoSync objects via the `spec.git.secretRef.name` field .
// The update to the Secret object will trigger a reconciliation of the RepoSync objects.
func (r *RepoSyncReconciler) mapSecretToRepoSyncs(secret client.Object) []reconcile.Request {
	// map the copied ns-reconciler Secret in the config-management-system to RepoSync request.
	if secret.GetNamespace() == configsync.ControllerNamespace {
		// Ignore secrets in the config-management-system namespace that don't start with ns-reconciler.
		if !strings.HasPrefix(secret.GetName(), reconciler.NsReconcilerPrefix) {
			return nil
		}
		allRepoSyncs := &v1beta1.RepoSyncList{}
		if err := r.client.List(context.Background(), allRepoSyncs); err != nil {
			klog.Error("failed to list all RepoSyncs for object (name: %s, namespace: %s): %v", secret.GetName(), secret.GetNamespace(), err)
			return nil
		}
		for _, rs := range allRepoSyncs.Items {
			// It is a one-to-one mapping between the copied ns-reconciler Secret and the RepoSync object,
			// so requeue the mapped RepoSync object and then return.
			reconcilerName := reconciler.NsReconcilerName(rs.GetNamespace(), rs.GetName())
			isGitSecret := secret.GetName() == ReconcilerResourceName(reconcilerName, rs.Spec.SecretRef.Name)
			if isGitSecret {
				return requeueRepoSyncRequest(secret, &rs)
			}
			isSAToken := strings.HasPrefix(secret.GetName(), reconcilerName+"-token-")
			if isSAToken {
				objectKey := client.ObjectKey{
					Name:      reconcilerName,
					Namespace: configsync.ControllerNamespace,
				}
				serviceAccount := &corev1.ServiceAccount{}
				if err := r.client.Get(context.Background(), objectKey, serviceAccount); err != nil {
					klog.Error("failed to get ServiceAccount %q in the %q namespace: %v", secret.GetName(), secret.GetNamespace(), err)
					return nil
				}
				for _, s := range serviceAccount.Secrets {
					if s.Name == secret.GetName() {
						return requeueRepoSyncRequest(secret, &rs)
					}
				}
			}
		}
		return nil
	}

	// map the user-managed ns-reconciler Secret in the RepoSync's namespace to RepoSync request.
	// The user-managed ns-reconciler Secret might be shared among multiple RepoSync objects in the same namespace,
	// so requeue all the attached RepoSync objects.
	attachedRepoSyncs := &v1beta1.RepoSyncList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(gitSecretRefField, secret.GetName()),
		Namespace:     secret.GetNamespace(),
	}
	if err := r.client.List(context.Background(), attachedRepoSyncs, listOps); err != nil {
		klog.Error("failed to list attached RepoSyncs for secret (name: %s, namespace: %s): %v", secret.GetName(), secret.GetNamespace(), err)
		return nil
	}

	requests := make([]reconcile.Request, len(attachedRepoSyncs.Items))
	attachedRSNames := make([]string, len(attachedRepoSyncs.Items))
	for i, rs := range attachedRepoSyncs.Items {
		attachedRSNames[i] = rs.GetName()
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      rs.GetName(),
				Namespace: rs.GetNamespace(),
			},
		}
	}
	if len(requests) > 0 {
		klog.Infof("Changes to Secret (name: %s, namespace: %s) triggers a reconciliation for the RepoSync object %q in the same namespace.", secret.GetName(), secret.GetNamespace(), strings.Join(attachedRSNames, ", "))
	}
	return requests
}

// mapObjectToRepoSync define a mapping from an object in 'config-management-system'
// namespace to a RepoSync to be reconciled.
func (r *RepoSyncReconciler) mapObjectToRepoSync(obj client.Object) []reconcile.Request {
	// Ignore changes from other namespaces because all the generated resources
	// exist in the config-management-system namespace.
	if obj.GetNamespace() != configsync.ControllerNamespace {
		return nil
	}

	// Ignore changes from resources without the ns-reconciler prefix or configsync.gke.io:ns-reconciler
	// because all the generated resources have the prefix.
	nsRoleBindingName := RepoSyncPermissionsName()
	if !strings.HasPrefix(obj.GetName(), reconciler.NsReconcilerPrefix) && obj.GetName() != nsRoleBindingName {
		return nil
	}

	allRepoSyncs := &v1beta1.RepoSyncList{}
	if err := r.client.List(context.Background(), allRepoSyncs); err != nil {
		klog.Error("failed to list all RepoSyncs for object (name: %s, namespace: %s): %v", obj.GetName(), obj.GetNamespace(), err)
		return nil
	}

	// Most of the resources are mapped to a single RepoSync object except RoleBinding.
	// All RepoSync objects share the same RoleBinding object, so requeue all RepoSync objects the RoleBinding is changed.
	// For other resources, requeue the mapping RepoSync object and then return.
	var requests []reconcile.Request
	var attachedRSNames []string
	for _, rs := range allRepoSyncs.Items {
		reconcilerName := reconciler.NsReconcilerName(rs.GetNamespace(), rs.GetName())
		switch obj.(type) {
		case *corev1.ConfigMap:
			suffix := strings.TrimPrefix(obj.GetName(), reconcilerName+"-")
			if suffix == reconcilermanager.GitSync ||
				suffix == reconcilermanager.HydrationController ||
				suffix == reconcilermanager.Reconciler {
				return requeueRepoSyncRequest(obj, &rs)
			}
		case *rbacv1.RoleBinding:
			if obj.GetName() == nsRoleBindingName {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      rs.GetName(),
						Namespace: rs.GetNamespace(),
					}})
				attachedRSNames = append(attachedRSNames, rs.GetName())
			}
		default: // Deployment and ServiceAccount
			if obj.GetName() == reconcilerName {
				return requeueRepoSyncRequest(obj, &rs)
			}
		}
	}
	if len(requests) > 0 {
		klog.Infof("Changes to %s (name: %s, namespace: %s) triggers a reconciliation for the RepoSync objects %q in the same namespace.",
			obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), obj.GetNamespace(), strings.Join(attachedRSNames, ", "))
	}
	return requests
}

func requeueRepoSyncRequest(obj client.Object, rs *v1beta1.RepoSync) []reconcile.Request {
	klog.Infof("Changes to %s (name: %s, namespace: %s) triggers a reconciliation for the RepoSync object %q in the same namespace.",
		obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), obj.GetNamespace(), rs.GetName())
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      rs.GetName(),
				Namespace: rs.GetNamespace(),
			},
		},
	}
}

func (r *RepoSyncReconciler) repoConfigMapMutations(ctx context.Context, rs *v1beta1.RepoSync, reconcilerName string) []configMapMutation {
	return []configMapMutation{
		{
			cmName: ReconcilerResourceName(reconcilerName, reconcilermanager.GitSync),
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
			cmName: ReconcilerResourceName(reconcilerName, reconcilermanager.HydrationController),
			data:   hydrationData(&rs.Spec.Git, declared.Scope(rs.Namespace), reconcilerName, r.hydrationPollingPeriod.String()),
		},
		{
			cmName: ReconcilerResourceName(reconcilerName, reconcilermanager.Reconciler),
			data:   reconcilerData(r.clusterName, rs.Name, reconcilerName, declared.Scope(rs.Namespace), &rs.Spec.Git, r.reconcilerPollingPeriod.String()),
		},
	}
}

// validateNamespaceSecret verify that any necessary Secret is present before creating ConfigMaps and Deployments.
func (r *RepoSyncReconciler) validateNamespaceSecret(ctx context.Context, repoSync *v1beta1.RepoSync) error {
	if SkipForAuth(repoSync.Spec.Auth) {
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

func (r *RepoSyncReconciler) validateNamespaceName(namespaceName string) error {
	if namespaceName == configsync.ControllerNamespace {
		return fmt.Errorf("RepoSync objects are not allowed in the %s namespace", configsync.ControllerNamespace)
	}
	return nil
}

func (r *RepoSyncReconciler) upsertRoleBinding(ctx context.Context, namespace string) error {
	var childRB rbacv1.RoleBinding
	childRB.Name = RepoSyncPermissionsName()
	childRB.Namespace = namespace

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childRB, func() error {
		return r.mutateRoleBinding(ctx, namespace, &childRB)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("RoleBinding successfully reconciled", operationSubjectName, childRB.Name, executedOperation, op)
	}
	return nil
}

func (r *RepoSyncReconciler) mutateRoleBinding(ctx context.Context, namespace string, rb *rbacv1.RoleBinding) error {
	// Update rolereference.
	rb.RoleRef = rolereference(RepoSyncPermissionsName(), "ClusterRole")

	reposyncList := &v1beta1.RepoSyncList{}
	if err := r.client.List(ctx, reposyncList, client.InNamespace(namespace)); err != nil {
		return errors.Wrapf(err, "failed to list the RepoSync objects in namespace %q", namespace)
	}
	return r.updateRoleBindingSubjects(rb, reposyncList)
}

func (r *RepoSyncReconciler) updateStatus(ctx context.Context, rs *v1beta1.RepoSync, log logr.Logger) error {
	rs.Status.ObservedGeneration = rs.Generation
	err := r.client.Status().Update(ctx, rs)
	if err != nil {
		log.Error(err, "failed to update RepoSync status")
	}
	return err
}

func (r *RepoSyncReconciler) mutationsFor(ctx context.Context, rs v1beta1.RepoSync, configMapDataHash []byte) mutateFn {
	return func(obj client.Object) error {
		d, ok := obj.(*appsv1.Deployment)
		if !ok {
			return errors.Errorf("expected appsv1 Deployment, got: %T", obj)
		}
		reconcilerName := reconciler.NsReconcilerName(rs.Namespace, rs.Name)
		// Mutate Annotation with the hash of configmap.data from all the ConfigMap
		// reconciler creates/updates.
		core.SetAnnotation(&d.Spec.Template, metadata.ConfigMapAnnotationKey, fmt.Sprintf("%x", configMapDataHash))
		// Add unique reconciler label
		core.SetLabel(&d.Spec.Template, metadata.ReconcilerLabel, reconcilerName)
		core.SetLabel(&d.Spec.Template, metadata.SyncNameLabel, rs.Name)
		core.SetLabel(&d.Spec.Template, metadata.SyncNamespaceLabel, rs.Namespace)
		templateSpec := &d.Spec.Template.Spec
		// Update ServiceAccountName. eg. ns-reconciler-<namespace>
		templateSpec.ServiceAccountName = reconcilerName
		// The Deployment object fetched from the API server has the field defined.
		// Update DeprecatedServiceAccount to avoid discrepancy in equality check.
		templateSpec.DeprecatedServiceAccount = reconcilerName
		// Mutate secret.secretname to secret reference specified in RepoSync CR.
		// Secret reference is the name of the secret used by git-sync container to
		// authenticate with the git repository using the authorization method specified
		// in the RepoSync CR.
		secretName := ReconcilerResourceName(reconcilerName, rs.Spec.SecretRef.Name)
		templateSpec.Volumes = filterVolumes(templateSpec.Volumes, rs.Spec.Auth, secretName)
		var updatedContainers []corev1.Container
		// Mutate spec.Containers to update name, configmap references and volumemounts.
		for _, container := range templateSpec.Containers {
			switch container.Name {
			case reconcilermanager.Reconciler:
				configmapRef := make(map[string]*bool)
				configmapRef[ReconcilerResourceName(reconcilerName, reconcilermanager.Reconciler)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(NamespaceReconcilerType))
			case reconcilermanager.HydrationController:
				configmapRef := make(map[string]*bool)
				configmapRef[ReconcilerResourceName(reconcilerName, reconcilermanager.HydrationController)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(NamespaceReconcilerType))
			case reconcilermanager.GitSync:
				configmapRef := make(map[string]*bool)
				configmapRef[ReconcilerResourceName(reconcilerName, reconcilermanager.GitSync)] = pointer.BoolPtr(false)
				container.EnvFrom = envFromSources(configmapRef)
				// Don't mount git-creds volume if auth is 'none' or 'gcenode'.
				container.VolumeMounts = volumeMounts(rs.Spec.Auth,
					container.VolumeMounts)
				// Update Environment variables for `token` Auth, which
				// passes the credentials as the Username and Password.
				if authTypeToken(rs.Spec.Auth) {
					container.Env = gitSyncTokenAuthEnv(secretName)
				}
				keys := GetKeys(ctx, r.client, rs.Spec.SecretRef.Name, rs.Namespace)
				container.Env = append(container.Env, gitSyncHTTPSProxyEnv(secretName, keys)...)
				mutateContainerResource(ctx, &container, rs.Spec.Override, string(NamespaceReconcilerType))
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
		// a constant) to the reconciler Deployment if auth: gcenode or auth: gcpserveraccount.
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
