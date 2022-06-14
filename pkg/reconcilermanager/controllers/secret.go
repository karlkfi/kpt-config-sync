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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	"kpt.dev/configsync/pkg/metadata"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func shouldUpsertGitSecret(rs *v1beta1.RepoSync) bool {
	return rs.Spec.SourceType == string(v1beta1.GitSource) && !SkipForAuth(rs.Spec.Auth)
}
func shouldUpsertHelmSecret(rs *v1beta1.RepoSync) bool {
	return rs.Spec.SourceType == string(v1beta1.HelmSource) && !SkipForAuth(rs.Spec.Helm.Auth)
}

// upsertSecret creates or updates the secret in config-management-system
// namespace using the existing secret in the reposync.namespace.
func upsertSecret(ctx context.Context, rs *v1beta1.RepoSync, c client.Client, reconcilerKey types.NamespacedName) (controllerutil.OperationResult, error) {
	// Secret is only created if sourceType is git or helm and auth is not 'none', 'gcenode', or 'gcpserviceaccount'.
	if !shouldUpsertGitSecret(rs) && !shouldUpsertHelmSecret(rs) {
		return controllerutil.OperationResultNone, nil
	}
	// namespaceSecret represent secret in reposync.namespace.
	namespaceSecret := &corev1.Secret{}
	var namespaceSecretName string
	if v1beta1.SourceType(rs.Spec.SourceType) == v1beta1.GitSource {
		namespaceSecretName = rs.Spec.SecretRef.Name
	} else {
		namespaceSecretName = rs.Spec.Helm.SecretRef.Name
	}
	if err := get(ctx, namespaceSecretName, rs.Namespace, namespaceSecret, c); err != nil {
		if apierrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, errors.Errorf(
				"%s not found. Create %s secret in %s namespace", namespaceSecretName, namespaceSecretName, rs.Namespace)
		}
		return controllerutil.OperationResultNone, errors.Wrapf(err, "error while retrieving namespace secret")
	}

	var reconcilerSecret corev1.Secret
	reconcilerSecret.Name = ReconcilerResourceName(reconcilerKey.Name, namespaceSecretName)
	reconcilerSecret.Namespace = reconcilerKey.Namespace

	return controllerruntime.CreateOrUpdate(ctx, c, &reconcilerSecret, func() error {
		labels := reconcilerSecret.Labels
		if labels == nil {
			labels = make(map[string]string, 2)
		}
		labels[metadata.SyncNamespaceLabel] = rs.Namespace
		labels[metadata.SyncNameLabel] = rs.Name
		reconcilerSecret.Labels = labels

		reconcilerSecret.Data = namespaceSecret.Data
		reconcilerSecret.Type = namespaceSecret.Type
		return nil
	})
}

// GetKeys returns the keys that are contained in the Secret.
func GetKeys(ctx context.Context, c client.Client, secretName, namespace string) map[string]bool {
	// namespaceSecret represent secret in reposync.namespace.
	namespaceSecret := &corev1.Secret{}
	if err := get(ctx, secretName, namespace, namespaceSecret, c); err != nil {
		return nil
	}
	results := map[string]bool{}
	for k := range namespaceSecret.Data {
		results[k] = true
	}
	return results
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

// SkipForAuth returns true if the passed auth is either 'none' or 'gcenode' or
// 'gcpserviceaccount'.
func SkipForAuth(auth configsync.AuthType) bool {
	switch auth {
	case configsync.AuthNone, configsync.AuthGCENode, configsync.AuthGCPServiceAccount:
		return true
	default:
		return false
	}
}
