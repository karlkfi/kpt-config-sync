package controllers

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// repoSyncResourceName returns name in the format ns-reconciler-<namespace>-<resourcename>.
func repoSyncResourceName(namespace, resourceName string) string {
	return fmt.Sprintf("%s-%s", reconciler.RepoSyncName(namespace), resourceName)
}

var prefix string = reconciler.RepoSyncPrefix + "-"

// nsOfReconciler return namespace by parsing namespace controller resource name.
func nsOfReconciler(obj client.Object) string {
	name := obj.GetName()
	if !strings.HasPrefix(name, prefix) {
		return ""
	}

	if _, ok := obj.(*corev1.ConfigMap); ok {
		return getNSFromConfigMap(name)
	}

	if _, ok := obj.(*corev1.Secret); ok {
		return getNSFromSecret(name)
	}

	// For all the other non-RepoSync object types registered in RepoSyncReconciler.SetupWithManager,
	// which currently includes Deployment, ServiceAccount, and RoleBinding.
	return strings.TrimPrefix(name, prefix)
}

func getNSFromSecret(name string) string {
	name = strings.TrimPrefix(name, prefix)
	sshKeySuffix := "-ssh-key"
	if strings.HasSuffix(name, sshKeySuffix) {
		name = trimSuffixes(name, sshKeySuffix)
	}

	if name != "" {
		// If the object name is in the format of "ns-reconciler-<ns>-token-xxxx"
		tokenSeparator := "-token-"
		name = strings.Split(name, tokenSeparator)[0]
	}
	return name
}

func getNSFromConfigMap(name string) string {
	name = strings.TrimPrefix(name, prefix)
	gitSyncSuffix := "-" + reconcilermanager.GitSync
	reconcilerSufix := "-" + reconcilermanager.Reconciler
	return trimSuffixes(name, gitSyncSuffix, reconcilerSufix)
}

func trimSuffixes(name string, opts ...string) string {
	for _, opt := range opts {
		name = strings.TrimSuffix(name, opt)
	}
	return name
}

// rootSyncResourceName returns name in the format root-reconciler-<resourcename>.
func rootSyncResourceName(resourceName string) string {
	return fmt.Sprintf("%s-%s", reconciler.RootSyncName, resourceName)
}

// repoSyncPermissionsName returns namespace reconciler permissions name.
// e.g. configsync.gke.io:ns-reconciler
func repoSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, reconciler.RepoSyncPrefix)
}

// rootSyncPermissionsName returns root reconciler permissions name.
// e.g. configsync.gke.io:root-reconciler
func rootSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, reconciler.RootSyncName)
}
