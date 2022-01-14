package controllers

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcilerResourceName returns resource name in the format <reconciler-name>-<resource-name>.
func ReconcilerResourceName(reconcilerName, resourceName string) string {
	return fmt.Sprintf("%s-%s", reconcilerName, resourceName)
}

var prefix = reconciler.NsReconcilerPrefix + "-"

// nsOfReconciler return namespace by parsing namespace controller resource name.
func nsOfReconciler(obj client.Object) string {
	name := obj.GetName()
	if !strings.HasPrefix(name, prefix) {
		return ""
	}

	if _, ok := obj.(*corev1.ConfigMap); ok {
		return getNSFromConfigMap(name)
	}

	if secret, ok := obj.(*corev1.Secret); ok {
		ns := core.GetAnnotation(secret, NSReconcilerNSAnnotationKey)
		if ns != "" {
			return ns
		}
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

// RepoSyncPermissionsName returns namespace reconciler permissions name.
// e.g. configsync.gke.io:ns-reconciler
func RepoSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, reconciler.NsReconcilerPrefix)
}

// RootSyncPermissionsName returns root reconciler permissions name.
// e.g. configsync.gke.io:root-reconciler
func RootSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, reconciler.RootReconcilerPrefix)
}
