package v1

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/constants"
)

const (
	// ConfigManagementPrefix is the prefix for all Nomos annotations and labels.
	ConfigManagementPrefix = configmanagement.GroupName + "/"

	// ClusterNameAnnotationKey is the annotation key set on Nomos-managed resources that refers to
	// the name of the cluster that the selectors are applied for.
	ClusterNameAnnotationKey = ConfigManagementPrefix + "cluster-name"

	// LegacyClusterSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to the name of the ClusterSelector resource.
	LegacyClusterSelectorAnnotationKey = ConfigManagementPrefix + "cluster-selector"

	// NamespaceSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to name of NamespaceSelector resource.
	NamespaceSelectorAnnotationKey = ConfigManagementPrefix + "namespace-selector"

	// DeclaredConfigAnnotationKey is the annotation key that stores the declared configuration of
	// a resource in Git.
	DeclaredConfigAnnotationKey = ConfigManagementPrefix + "declared-config"

	// SourcePathAnnotationKey is the annotation key representing the relative path from POLICY_DIR
	// where the object was originally declared. Paths are slash-separated and OS-agnostic.
	SourcePathAnnotationKey = ConfigManagementPrefix + "source-path"

	// SyncTokenAnnotationKey is the annotation key representing the last version token that a Nomos-
	// managed resource was successfully synced from.
	SyncTokenAnnotationKey = ConfigManagementPrefix + "token"

	// ResourceManagementKey is the annotation that indicates if Nomos will manage the content and
	// lifecycle for the resource.
	ResourceManagementKey = ConfigManagementPrefix + "managed"
	// ResourceManagementEnabled is the value corresponding to ResourceManagementKey indicating that
	// Nomos will manage content and lifecycle for the given resource.
	ResourceManagementEnabled = "enabled"
	// ResourceManagementDisabled is the value corresponding to ResourceManagementKey indicating that
	// Nomos will not manage content and lifecycle for the given resource.
	// By design, the `configmanagement.gke.io/managed: disabled` annotation
	// should not be pushed to the cluster. Instead, we remove all the Config
	// Sync metadata from the object on the cluster.
	ResourceManagementDisabled = "disabled"

	// The following annotations implement the extended resource status specification.

	// ResourceStatusErrorsKey is the annotation that indicates any errors, encoded as a JSON array.
	ResourceStatusErrorsKey = ConfigManagementPrefix + "errors"

	// ResourceStatusReconcilingKey is the annotation that indicates reasons why a resource is
	// reconciling, encoded as a JSON array.
	ResourceStatusReconcilingKey = ConfigManagementPrefix + "reconciling"
)

// SyncerAnnotations returns the set of Nomos annotations that the syncer should manage.
func SyncerAnnotations() []string {
	return []string{
		ClusterNameAnnotationKey,
		LegacyClusterSelectorAnnotationKey,
		constants.ClusterNameSelectorAnnotationKey,
		NamespaceSelectorAnnotationKey,
		DeclaredConfigAnnotationKey,
		SourcePathAnnotationKey,
		SyncTokenAnnotationKey,
		ResourceManagementKey,
		ResourceStatusErrorsKey,
		ResourceStatusReconcilingKey,
	}
}
