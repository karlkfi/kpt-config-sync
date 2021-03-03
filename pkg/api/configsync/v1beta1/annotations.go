package v1beta1

import "github.com/google/nomos/pkg/api/configsync"

const (
	// ConfigSyncPrefix is the prefix for all ConfigSync annotations and labels.
	ConfigSyncPrefix = configsync.GroupName + "/"

	// ConfigMapAnnotationKey is the annotation key representing the hash of all the configmaps
	// required to run root reconciler and namespace reconciler process.
	ConfigMapAnnotationKey = ConfigSyncPrefix + "configmap"

	// DeclaredFieldsKey is the annotation key that stores the declared configuration of
	// a resource in Git. This uses the same format as the managed fields of server-side apply.
	DeclaredFieldsKey = ConfigSyncPrefix + "declared-fields"

	// GitContextKey is the annotation key for the git source-of-truth a resource is synced from.
	GitContextKey = ConfigSyncPrefix + "git-context"

	// ResourceManagerKey is the annotation that indicates which multi-repo reconciler is managing
	// the resource.
	ResourceManagerKey = ConfigSyncPrefix + "manager"

	// ClusterNameSelectorAnnotationKey is the annotation key set on ConfigSync-managed resources that refers
	// to the name of the ClusterSelector resource.
	ClusterNameSelectorAnnotationKey = ConfigSyncPrefix + "cluster-name-selector"
)
