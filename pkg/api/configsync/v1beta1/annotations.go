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

	// LifecyclePrefix is the prefix for all lifecycle annotations.
	LifecyclePrefix = "client.lifecycle.config.k8s.io"

	// LifecycleMutationAnnotation is the lifecycle annotation key for the mutation
	// operation. The annotation must be declared in the repository in order to
	// function properly. This annotation only has effect when the object
	// updated in the cluster or the declaration changes. It has no impact on
	// behavior related to object creation/deletion, or if the object does not
	// already exist.
	LifecycleMutationAnnotation = LifecyclePrefix + "/mutation"

	// IgnoreMutation is the value used with LifecycleMutationAnnotation to
	// prevent mutating a resource. That is, if the resource exists on the cluster
	// then ACM will make no attempt to modify it.
	IgnoreMutation = "ignore"

	// ResourceIDKey is the annotation that indicates the resource's GKNN.
	ResourceIDKey = ConfigSyncPrefix + "resource-id"
)

// ConfigSyncAnnotations contain the keys for ConfigSync annotations.
var ConfigSyncAnnotations = []string{
	DeclaredFieldsKey,
	GitContextKey,
	ResourceManagerKey,
	ResourceIDKey,
}
