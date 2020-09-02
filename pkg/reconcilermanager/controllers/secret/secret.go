package secret

import (
	"context"
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if err := get(ctx, RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name),
		v1.NSConfigManagementSystem, existingsecret, c); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err,
				"failed to get secret %s in namespace %s", RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name), v1.NSConfigManagementSystem)
		}
		// Secret not present in config-management-system namespace. Create one using
		// secret in reposync.namespace.
		if err := create(ctx, rs, namespaceSecret, c); err != nil {
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
func create(ctx context.Context, reposync *v1alpha1.RepoSync, namespaceSecret *corev1.Secret, c client.Client) error {
	newSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.Secret().Kind,
			APIVersion: kinds.Secret().Version,
		},
	}

	// Set OwnerReferences, so that when the RepoSync CustomResource is deleted,
	// the Secret is also deleted.
	newSecret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         v1.SchemeGroupVersion.String(),
			Kind:               reposync.Kind,
			Name:               reposync.Name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                reposync.UID,
		},
	}

	// mutate newSecret with values from the secret in reposync.namespace.
	newSecret.Name = RepoSyncSecretName(namespaceSecret.Namespace, namespaceSecret.Name)
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

const (
	// RepoSyncSecret represet suffix used for reposync secret name in
	// config-management-system.
	RepoSyncSecret = "reposync-secret"
)

// RepoSyncSecretName return name of the reposync secret eg.
// reposync-secret-<namespace>-<name>
func RepoSyncSecretName(namespace, name string) string {
	prefix := []string{RepoSyncSecret}
	return strings.Join(append(prefix, namespace, name), "-")
}

// SkipForAuth returns true if the passed auth is either 'none' or 'gcenode'.
func SkipForAuth(auth string) bool {
	return auth == "none" || auth == "gcenode"
}
