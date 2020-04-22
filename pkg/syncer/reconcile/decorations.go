package reconcile

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
)

// SyncedAt marks the resource as synced at the passed sync token.
func SyncedAt(obj core.LabeledAndAnnotated, token string) {
	core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-manged.
func enableManagement(obj core.LabeledAndAnnotated) {
	core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
}

// removeNomosMeta removes syncer-managed Nomos system annotations and labels from the given resource.
// The resource is modified in place.
func removeNomosMeta(obj core.LabeledAndAnnotated) {
	core.RemoveAnnotations(obj, v1.SyncerAnnotations()...)
	core.RemoveLabels(obj, v1.SyncerLabels())
}
