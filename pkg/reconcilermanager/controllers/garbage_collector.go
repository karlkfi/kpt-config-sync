package controllers

import (
	"context"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Cleanup namespace controller resources when reposync is not found since,
// we dont leverage ownerRef because namespace controller resources are
// created in config-management-system namespace instead of resposync namespace.
//
// NOTE: Update this method when resources created by namespace controller changes.
func (r *RepoSyncReconciler) cleanupNSControllerResources(ctx context.Context, ns string) error {
	r.log.Info("Cleaning up namespace controller resources", "namespace", ns)

	// Delete namespace controller resources and return to reconcile loop in case
	// of errors to try cleaning up resources again.

	// Deployment
	if err := r.deleteDeployment(ctx, ns); err != nil {
		return err
	}
	// configmaps
	if err := r.deleteConfigmap(ctx, ns); err != nil {
		return err
	}
	// serviceaccount
	if err := r.deleteServiceAccount(ctx, ns); err != nil {
		return err
	}
	// rolebinding
	if err := r.deleteRoleBinding(ctx, ns); err != nil {
		return err
	}
	// secret
	return r.deleteSecret(ctx, ns)
}

func (r *RepoSyncReconciler) cleanup(ctx context.Context, name, namespace string, gvk schema.GroupVersionKind) error {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetGroupVersionKind(gvk)
	err := r.client.Delete(ctx, u)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.log.V(4).Info("resource not present", "namespace", namespace, "resource", name)
			return nil
		}
	}
	return err
}

func (r *RepoSyncReconciler) deleteSecret(ctx context.Context, namespace string) error {
	secretList := &corev1.SecretList{}
	if err := r.client.List(ctx, secretList, client.InNamespace(configsync.ControllerNamespace)); err != nil {
		return err
	}

	// nsPrefix specifies reposync secret prefix 'ns-reconciler-<namespace>'
	nsPrefix := reconciler.RepoSyncName(namespace)

	for _, s := range secretList.Items {
		if strings.HasPrefix(s.Name, nsPrefix) {
			if err := r.client.Delete(ctx, &s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *RepoSyncReconciler) deleteConfigmap(ctx context.Context, namespace string) error {
	cms := []string{
		RepoSyncResourceName(namespace, reconcilermanager.Reconciler),
		RepoSyncResourceName(namespace, reconcilermanager.HydrationController),
		RepoSyncResourceName(namespace, reconcilermanager.GitSync),
	}
	for _, c := range cms {
		if err := r.cleanup(ctx, c, v1.NSConfigManagementSystem, kinds.ConfigMap()); err != nil {
			return err
		}
	}
	return nil
}

func (r *RepoSyncReconciler) deleteServiceAccount(ctx context.Context, namespace string) error {
	saName := reconciler.RepoSyncName(namespace)
	return r.cleanup(ctx, saName, v1.NSConfigManagementSystem, kinds.ServiceAccount())
}

func (r *RepoSyncReconciler) deleteRoleBinding(ctx context.Context, namespace string) error {
	rbName := RepoSyncPermissionsName()
	return r.cleanup(ctx, rbName, namespace, kinds.RoleBinding())
}

func (r *RepoSyncReconciler) deleteDeployment(ctx context.Context, namespace string) error {
	dpName := reconciler.RepoSyncName(namespace)
	return r.cleanup(ctx, dpName, v1.NSConfigManagementSystem, kinds.Deployment())
}

func (r *RootSyncReconciler) deleteClusterRoleBinding(ctx context.Context) error {
	crbName := RootSyncPermissionsName()
	return r.cleanup(ctx, crbName, kinds.ClusterRoleBinding())
}

// cleanup cleans up cluster-scoped resources that are created for RootSync.
// Other namespace-scoped resources are garbage collected via OwnerReferences.
// Cluster-scoped resources cannot be handled via OwnerReferences because
// RootSync is namespace-scoped, and cluster-scoped dependents can only specify
// cluster-scoped owners.
func (r *RootSyncReconciler) cleanup(ctx context.Context, name string, gvk schema.GroupVersionKind) error {
	u := &unstructured.Unstructured{}
	u.SetName(name)
	u.SetGroupVersionKind(gvk)
	err := r.client.Delete(ctx, u)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.log.V(4).Info("cluster-scoped resource not present", "resource", name)
			return nil
		}
	}
	return err
}
