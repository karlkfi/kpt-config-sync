package metadata

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CommonAnnotationKeys include the annotation keys used in both the mono-repo and multi-repo mode.
var CommonAnnotationKeys = []string{
	v1.ClusterNameAnnotationKey,
	v1.ResourceManagementKey,
	v1.SourcePathAnnotationKey,
	v1.SyncTokenAnnotationKey,
	constants.DeclaredFieldsKey,
	constants.ResourceIDKey,
}

// MultiRepoOnlyAnnotationKeys include the annotation keys used only in the multi-repo mode.
var MultiRepoOnlyAnnotationKeys = []string{
	constants.GitContextKey,
	constants.ResourceManagerKey,
	constants.OwningInventoryKey,
}

// GetNomosAnnotationKeys returns the set of Nomos annotations that Config Sync should manage.
func GetNomosAnnotationKeys(multiRepo bool) []string {
	if multiRepo {
		return append(CommonAnnotationKeys, MultiRepoOnlyAnnotationKeys...)
	}
	return CommonAnnotationKeys
}

// sourceAnnotations is a map of annotations that are valid to exist on objects
// in the source repository.
// These annotations are set by Config Sync users.
var sourceAnnotations = map[string]bool{
	v1.NamespaceSelectorAnnotationKey:          true,
	v1.LegacyClusterSelectorAnnotationKey:      true,
	constants.ClusterNameSelectorAnnotationKey: true,
	v1.ResourceManagementKey:                   true,
	constants.LifecycleMutationAnnotation:      true,
}

// IsSourceAnnotation returns true if the annotation is a ConfigSync source
// annotation.
func IsSourceAnnotation(k string) bool {
	return sourceAnnotations[k]
}

// HasConfigSyncPrefix returns true if the string begins with a ConfigSync
// annotation prefix.
func HasConfigSyncPrefix(s string) bool {
	return strings.HasPrefix(s, v1.ConfigManagementPrefix) || strings.HasPrefix(s, constants.ConfigSyncPrefix)
}

// IsConfigSyncAnnotationKey returns whether an annotation key is a Config Sync annotation key.
func IsConfigSyncAnnotationKey(k string) bool {
	return HasConfigSyncPrefix(k) ||
		strings.HasPrefix(k, constants.LifecycleMutationAnnotation) ||
		k == constants.OwningInventoryKey
}

// isConfigSyncAnnotation returns whether an annotation is a Config Sync annotation.
func isConfigSyncAnnotation(k, v string) bool {
	return IsConfigSyncAnnotationKey(k) || (k == hnc.AnnotationKeyV1A2 && v == configmanagement.GroupName)
}

// IsConfigSyncLabelKey returns whether a label key is a Config Sync label key.
func IsConfigSyncLabelKey(k string) bool {
	return HasConfigSyncPrefix(k) || k == v1.ManagedByKey
}

// isConfigSyncLabel returns whether a label is a Config Sync label.
func isConfigSyncLabel(k, v string) bool {
	return HasConfigSyncPrefix(k) || (k == v1.ManagedByKey && v == v1.ManagedByValue)
}

// HasConfigSyncMetadata returns true if the given obj has at least one Config Sync annotation or label.
func HasConfigSyncMetadata(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	for k, v := range annotations {
		if isConfigSyncAnnotation(k, v) {
			return true
		}
	}

	labels := obj.GetLabels()
	for k, v := range labels {
		if isConfigSyncLabel(k, v) {
			return true
		}
	}
	return false
}

// RemoveConfigSyncMetadata removes the Config Sync metadata, including both Config Sync
// annotations and labels, from the given resource.
// The only Config Sync metadata which will not be removed is `constants.LifecycleMutationAnnotation`.
// The resource is modified in place. Returns true if the object was modified.
func RemoveConfigSyncMetadata(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	labels := obj.GetLabels()
	before := len(annotations) + len(labels)

	// Remove Config Sync annotations
	for k, v := range annotations {
		if isConfigSyncAnnotation(k, v) && k != constants.LifecycleMutationAnnotation {
			delete(annotations, k)
		}
	}
	obj.SetAnnotations(annotations)

	// Remove Config Sync labels
	for k, v := range labels {
		if isConfigSyncLabel(k, v) {
			delete(labels, k)
		}
	}
	obj.SetLabels(labels)

	after := len(obj.GetAnnotations()) + len(obj.GetLabels())
	return before != after
}
