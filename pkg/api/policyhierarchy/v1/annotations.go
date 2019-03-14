package v1

import (
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
)

const (
	// NomosPrefix is the prefix for all Nomos annotations.
	// TODO(125862145): use policyhierarchy.GroupName below after resource groups are updated to configmanagement.gke.io
	NomosPrefix = policyhierarchy.GroupName + "/"

	// ClusterNameAnnotationKey is the annotation key set on Nomos-managed resources that refers to
	// the name of the cluster that the selectors are applied for.
	ClusterNameAnnotationKey = NomosPrefix + "cluster-name"

	// ClusterSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to the name of the ClusterSelector resource.
	ClusterSelectorAnnotationKey = NomosPrefix + "cluster-selector"

	// NamespaceSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to name of NamespaceSelector resource.
	NamespaceSelectorAnnotationKey = NomosPrefix + "namespace-selector"

	// SourcePathAnnotationKey is the annotation key representing the relative path from POLICY_DIR
	// where the object was originally declared. Paths are slash-separated and OS-agnostic.
	SourcePathAnnotationKey = NomosPrefix + "source-path"

	// SyncTokenAnnotationKey is the annotation key representing the last version token that a Nomos-
	// managed resource was successfully synced from.
	SyncTokenAnnotationKey = NomosPrefix + "sync-token"

	// ResourceManagementKey indicates if Nomos will manage the content and lifecycle for the resource.
	ResourceManagementKey = NomosPrefix + "managed"

	// ResourceManagementValue is the value corresponding to ResourceManagementKey indicating that
	// Nomos will manage content and lifecycle for the given resource.
	ResourceManagementValue = "enabled"

	// ResourceManagementDisabledValue is the value corresponding to ResourceManagementKey indicating that
	// Nomos will not manage content and lifecycle for the given resource.
	ResourceManagementDisabledValue = "disabled"
)

// HasNomosPrefix returns true if the string begins with the Nomos annotation prefix.
func HasNomosPrefix(s string) bool {
	return strings.HasPrefix(s, NomosPrefix)
}

// inputAnnotations is a map of annotations that are valid to exist on objects when imported from
// the filesystem.
var inputAnnotations = map[string]bool{
	NamespaceSelectorAnnotationKey: true,
	ClusterSelectorAnnotationKey:   true,
}

// IsInputAnnotation returns true if the annotation is a Nomos input annotation.
func IsInputAnnotation(s string) bool {
	return inputAnnotations[s]
}

var nomosAnnotations = map[string]bool{
	ClusterNameAnnotationKey:       true,
	ClusterSelectorAnnotationKey:   true,
	NamespaceSelectorAnnotationKey: true,
	SourcePathAnnotationKey:        true,
	SyncTokenAnnotationKey:         true,
	ResourceManagementKey:          true,
}

// IsAnnotation returns true if the annotation is a nomos system annotation.
func IsAnnotation(a string) bool {
	return nomosAnnotations[a]
}

// GetClusterSelectorAnnotation returns the value of the cluster selector annotation
// among the given annotations.  If the annotation is not there, "" is returned.
func GetClusterSelectorAnnotation(a map[string]string) string {
	// Looking up in a nil map will also return "".
	return a[ClusterSelectorAnnotationKey]
}
