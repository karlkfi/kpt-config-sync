package controllers

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configsync"
)

// repoSyncName returns name in the format ns-reconciler-<namespace>.
func repoSyncName(namespace string) string {
	return fmt.Sprintf("%s-%s", repoSyncReconcilerPrefix, namespace)
}

// repoSyncResourceName returns name in the format ns-reconciler-<namespace>-<resourcename>.
func repoSyncResourceName(namespace, resourceName string) string {
	return fmt.Sprintf("%s-%s-%s", repoSyncReconcilerPrefix, namespace, resourceName)
}

// parseRepoSyncReconciler parses namespace reconciler deployment name ns-reconciler-<namespace>
// and returns namespace.
func parseRepoSyncReconciler(name string) string {
	prefix := repoSyncReconcilerPrefix + "-"
	if strings.HasPrefix(name, prefix) {
		return strings.TrimPrefix(name, prefix)
	}
	return ""
}

// rootSyncResourceName returns name in the format root-reconciler-<resourcename>.
func rootSyncResourceName(resourceName string) string {
	return fmt.Sprintf("%s-%s", rootSyncReconcilerName, resourceName)
}

// repoSyncPermissionsName returns namespace reconciler permissions name.
// e.g. configsync.gke.io:ns-reconciler
func repoSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, repoSyncReconcilerPrefix)
}

// rootSyncPermissionsName returns root reconciler permissions name.
// e.g. configsync.gke.io:root-reconciler
func rootSyncPermissionsName() string {
	return fmt.Sprintf("%s:%s", configsync.GroupName, rootSyncReconcilerName)
}
