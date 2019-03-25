package v1

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
)

const (
	// ConfigManagementPrefix is the prefix for all Nomos annotations.
	ConfigManagementPrefix = configmanagement.GroupName + "/"

	// ClusterNameAnnotationKey is the annotation key set on Nomos-managed resources that refers to
	// the name of the cluster that the selectors are applied for.
	ClusterNameAnnotationKey = ConfigManagementPrefix + "cluster-name"

	// ClusterSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to the name of the ClusterSelector resource.
	ClusterSelectorAnnotationKey = ConfigManagementPrefix + "cluster-selector"

	// NamespaceSelectorAnnotationKey is the annotation key set on Nomos-managed resources that refers
	// to name of NamespaceSelector resource.
	NamespaceSelectorAnnotationKey = ConfigManagementPrefix + "namespace-selector"

	// SourcePathAnnotationKey is the annotation key representing the relative path from POLICY_DIR
	// where the object was originally declared. Paths are slash-separated and OS-agnostic.
	SourcePathAnnotationKey = ConfigManagementPrefix + "source-path"

	// SyncTokenAnnotationKey is the annotation key representing the last version token that a Nomos-
	// managed resource was successfully synced from.
	SyncTokenAnnotationKey = ConfigManagementPrefix + "token"

	// ResourceManagementKey indicates if Nomos will manage the content and lifecycle for the resource.
	ResourceManagementKey = ConfigManagementPrefix + "managed"

	// ResourceManagementEnabled is the value corresponding to ResourceManagementKey indicating that
	// Nomos will manage content and lifecycle for the given resource.
	ResourceManagementEnabled = "enabled"

	// ResourceManagementDisabled is the value corresponding to ResourceManagementKey indicating that
	// Nomos will not manage content and lifecycle for the given resource.
	ResourceManagementDisabled = "disabled"
)

// HasConfigManagementPrefix returns true if the string begins with the Nomos annotation prefix.
func HasConfigManagementPrefix(s string) bool {
	return strings.HasPrefix(s, ConfigManagementPrefix)
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

// HasNomosAnnotation returns true if the given map has at least one Nomos annotation.
func HasNomosAnnotation(a map[string]string) bool {
	for k := range nomosAnnotations {
		if IsAnnotation(k) {
			return true
		}
	}
	return false
}

// GetClusterSelectorAnnotation returns the value of the cluster selector annotation
// among the given annotations.  If the annotation is not there, "" is returned.
func GetClusterSelectorAnnotation(a map[string]string) string {
	// Looking up in a nil map will also return "".
	return a[ClusterSelectorAnnotationKey]
}

// RemoveNomos removes Nomos system annotations from the given map.  The map is
// modified in place.
func RemoveNomos(a map[string]string) {
	if a == nil {
		return
	}
	for k := range nomosAnnotations {
		delete(a, k)
	}
}
