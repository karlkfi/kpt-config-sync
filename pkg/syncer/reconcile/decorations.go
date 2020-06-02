package reconcile

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
)

// SyncedAt marks the resource as synced at the passed sync token.
func SyncedAt(obj core.LabeledAndAnnotated, token string) {
	core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-managed.
func enableManagement(obj core.LabeledAndAnnotated) {
	core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
}

// RemoveNomosLabelsAndAnnotations removes syncer-managed Nomos system
// annotations and labels from the given resource. The resource is modified in
// place. Returns true if the object was modified.
func RemoveNomosLabelsAndAnnotations(obj core.LabeledAndAnnotated) bool {
	before := len(obj.GetAnnotations()) + len(obj.GetLabels())
	core.RemoveAnnotations(obj, append(v1.SyncerAnnotations(), hnc.AnnotationKey)...)
	core.RemoveLabels(obj, v1.SyncerLabels())
	after := len(obj.GetAnnotations()) + len(obj.GetLabels())
	return before != after
}
