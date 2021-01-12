package controllers

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RepoSyncName returns name in the format ns-reconciler-<namespace>.
func RepoSyncName(namespace string) string {
	return fmt.Sprintf("%s-%s", repoSyncReconcilerPrefix, namespace)
}

// repoSyncResourceName returns name in the format ns-reconciler-<namespace>-<resourcename>.
func repoSyncResourceName(namespace, resourceName string) string {
	return fmt.Sprintf("%s-%s-%s", repoSyncReconcilerPrefix, namespace, resourceName)
}

// parseRepoSyncReconciler parses namespace reconciler deployment name ns-reconciler-<namespace>
// and returns namespace.
func parseRepoSyncReconciler(name string, obj runtime.Object) string {
	prefix := repoSyncReconcilerPrefix + "-"
	var ns string
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	ns = strings.TrimPrefix(name, prefix)

	// If an obj is a ConfigMap then trim following suffix from the name of the
	// object.
	gitSyncSuffix := "-" + gitSync
	reconcilerSufix := "-" + reconciler
	if _, ok := obj.(*corev1.ConfigMap); ok {
		ns = trimConfigMapSuffix(ns, gitSyncSuffix, reconcilerSufix)
	}

	// If an obj is a Secret then trim following suffix from the name of the
	// object.
	sshKeySuffix := "-ssh-key"
	if _, ok := obj.(*corev1.Secret); ok {
		ns = trimConfigMapSuffix(ns, sshKeySuffix)
	}

	return ns
}

func trimConfigMapSuffix(name string, opts ...string) string {
	for _, opt := range opts {
		name = strings.TrimSuffix(name, opt)
	}
	return name
}

// rootSyncResourceName returns name in the format root-reconciler-<resourcename>.
func rootSyncResourceName(resourceName string) string {
	return fmt.Sprintf("%s-%s", RootSyncReconcilerName, resourceName)
}

// repoSyncPermissionsName returns namespace reconciler permissions name.
// e.g. configsync.gke.io:ns-reconciler
func repoSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, repoSyncReconcilerPrefix)
}

// rootSyncPermissionsName returns root reconciler permissions name.
// e.g. configsync.gke.io:root-reconciler
func rootSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, RootSyncReconcilerName)
}
