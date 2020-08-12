package controllers

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

// validateSecretExist validate the presence of secret in the cluster
func validateSecretExist(ctx context.Context, secretRef, namespace string, c client.Client) (*corev1.Secret, error) {
	secretNN := client.ObjectKey{
		Name:      secretRef,
		Namespace: namespace,
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, secretNN, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.Errorf(
				"%s not found. Create %s secret in %s namespace", secretRef, secretRef, namespace)
		}
		return nil, errors.Errorf("error while retrieving git-creds secret: %v", err)
	}
	return secret, nil
}

// validateSecretData verify secret data for the given auth type.
func validateSecretData(auth string, secret *corev1.Secret) error {
	switch auth {
	case "ssh":
		if _, ok := secret.Data["ssh"]; !ok {
			return fmt.Errorf("git secretType was set as \"ssh\" but ssh key is not present in %v secret", secret.Name)
		}
	case "cookiefile":
		if _, ok := secret.Data["cookie_file"]; !ok {
			return fmt.Errorf("git secretType was set as \"cookiefile\" but cookie_file key is not present in %v secret", secret.Name)
		}
	case "token":
		if _, ok := secret.Data["token"]; !ok {
			return fmt.Errorf("git secretType was set as \"token\" but token key is not present in %v secret", secret.Name)
		}
		if _, ok := secret.Data["username"]; !ok {
			return fmt.Errorf("git secretType was set as \"token\" but username key is not present in %v secret", secret.Name)
		}
	case "none":
	case "gcenode":
	default:
		return fmt.Errorf("git secretType is set to unsupported value: %q", auth)
	}
	return nil
}
