package secrets

import (
	"context"
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Put secret in config-management-system namespace using the
// existing secret in the reposync.namespace.
func Put(ctx context.Context, rs *v1alpha1.RepoSync, c client.Client) error {
	// Secret is only created if auth is not 'none' or 'gcenode'.
	if SkipForAuth(rs.Spec.Auth) {
		return nil
	}

	// namespaceSecret represent secret in reposync.namespace.
	namespaceSecret := &corev1.Secret{}
	if err := get(ctx, rs.Spec.SecretRef.Name, rs.Namespace, namespaceSecret, c); err != nil {
		if apierrors.IsNotFound(err) {
			return errors.Errorf(
				"%s not found. Create %s secret in %s namespace", rs.Spec.SecretRef.Name, rs.Spec.SecretRef.Name, rs.Namespace)
		}
		return errors.Wrapf(err, "error while retrieving namespace secret")
	}

	// existingsecret represent secret in config-management-system namespace.
	existingsecret := &corev1.Secret{}
	if err := get(ctx, NamespaceReconcilerSecretName(rs.Namespace, rs.Spec.SecretRef.Name),
		v1.NSConfigManagementSystem, existingsecret, c); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err,
				"failed to get secret %s in namespace %s", NamespaceReconcilerSecretName(rs.Namespace, rs.Spec.SecretRef.Name), v1.NSConfigManagementSystem)
		}
		// Secret not present in config-management-system namespace. Create one using
		// secret in reposync.namespace.
		if err := create(ctx, namespaceSecret, c); err != nil {
			return errors.Wrapf(err,
				"failed to create %s secret in %s namespace",
				rs.Spec.SecretRef.Name, v1.NSConfigManagementSystem)
		}
		return nil
	}
	// Update the existing secret in config-management-system.
	if err := update(ctx, existingsecret, namespaceSecret, c); err != nil {
		return errors.Wrapf(err, "failed to update the secret %s", existingsecret.Name)
	}
	return nil
}

// get secret using provided namespace and name.
func get(ctx context.Context, name, namespace string, secret *corev1.Secret, c client.Client) error {
	// NamespacedName for the secret.
	nn := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	return c.Get(ctx, nn, secret)
}

// create secret get the existing secret in reposync.namespace and use secret.data and
// secret.type to create a new secret in config-management-system namespace.
func create(ctx context.Context, namespaceSecret *corev1.Secret, c client.Client) error {
	newSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.Secret().Kind,
			APIVersion: kinds.Secret().Version,
		},
	}

	// mutate newSecret with values from the secret in reposync.namespace.
	newSecret.Name = NamespaceReconcilerSecretName(namespaceSecret.Namespace, namespaceSecret.Name)
	newSecret.Namespace = v1.NSConfigManagementSystem
	newSecret.Data = namespaceSecret.Data
	newSecret.Type = namespaceSecret.Type

	return c.Create(ctx, newSecret)
}

// update secret fetch the existing secret from the cluster and use secret.data and
// secret.type to create a new secret in config-management-system namespace.
func update(ctx context.Context, existingsecret *corev1.Secret, namespaceSecret *corev1.Secret, c client.Client) error {

	// Update data and type for the existing secret with values from the secret in
	// reposync.namespace
	existingsecret.Data = namespaceSecret.Data
	existingsecret.Type = namespaceSecret.Type

	return c.Update(ctx, existingsecret)
}

// NamespaceReconcilerSecretName return name of the Namespace reconciler secret.
// e.g. ns-reconciler-<namespace>-<name>
func NamespaceReconcilerSecretName(namespace, name string) string {
	return fmt.Sprintf("%s-%s", reconciler.RepoSyncName(namespace), name)
}

// SkipForAuth returns true if the passed auth is either 'none' or 'gcenode' or
// 'gcpserviceaccount'.
func SkipForAuth(auth string) bool {
	switch auth {
	case v1alpha1.GitSecretNone, v1alpha1.GitSecretGCENode, v1alpha1.GitSecretGCPServiceAccount:
		return true
	default:
		return false
	}
}
