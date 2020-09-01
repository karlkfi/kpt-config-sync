package controllers

import (
	"fmt"

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
