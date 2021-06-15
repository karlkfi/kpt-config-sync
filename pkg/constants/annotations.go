package constants

const (
	// ConfigMapAnnotationKey is the annotation key representing the hash of all the configmaps
	// required to run root reconciler and namespace reconciler process.
	// This annotation is set by Config Sync.
	ConfigMapAnnotationKey = ConfigSyncPrefix + "configmap"

	// DeclaredFieldsKey is the annotation key that stores the declared configuration of
	// a resource in Git. This uses the same format as the managed fields of server-side apply.
	// This annotation is set by Config Sync.
	DeclaredFieldsKey = ConfigSyncPrefix + "declared-fields"

	// GitContextKey is the annotation key for the git source-of-truth a resource is synced from.
	// This annotation is set by Config Sync.
	GitContextKey = ConfigSyncPrefix + "git-context"

	// ResourceManagerKey is the annotation that indicates which multi-repo reconciler is managing
	// the resource.
	// This annotation is set by Config Sync.
	ResourceManagerKey = ConfigSyncPrefix + "manager"

	// ClusterNameSelectorAnnotationKey is the annotation key set on ConfigSync-managed resources that refers
	// to the name of the ClusterSelector resource.
	// This annotation is set by Config Sync users.
	ClusterNameSelectorAnnotationKey = ConfigSyncPrefix + "cluster-name-selector"

	// LifecycleMutationAnnotation is the lifecycle annotation key for the mutation
	// operation. The annotation must be declared in the repository in order to
	// function properly. This annotation only has effect when the object
	// updated in the cluster or the declaration changes. It has no impact on
	// behavior related to object creation/deletion, or if the object does not
	// already exist.
	// This annotation is set by Config Sync users.
	LifecycleMutationAnnotation = LifecyclePrefix + "/mutation"

	// IgnoreMutation is the value used with LifecycleMutationAnnotation to
	// prevent mutating a resource. That is, if the resource exists on the cluster
	// then ACM will make no attempt to modify it.
	IgnoreMutation = "ignore"

	// ResourceIDKey is the annotation that indicates the resource's GKNN.
	// This annotation is set by Config Sync.
	ResourceIDKey = ConfigSyncPrefix + "resource-id"

	// OwningInventoryKey is the annotation key for marking the owning-inventory object.
	// This annotation is needed when the remediator needs to create a managed resource deleted by another entity.
	// To support this, we need to add this annotation to every managed resource.
	// Note: including this annotation in the resources to be applied by the kpt live apply library is not a problem.
	// This annotation is set by Config Sync.
	OwningInventoryKey = "config.k8s.io/owning-inventory"
)

// ConfigSyncAnnotations contain the keys for ConfigSync annotations.
var ConfigSyncAnnotations = []string{
	DeclaredFieldsKey,
	GitContextKey,
	ResourceManagerKey,
	ResourceIDKey,
}
