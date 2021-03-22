package reconcile

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncedAt marks the resource as synced at the passed sync token.
func SyncedAt(obj client.Object, token string) {
	core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-managed.
func enableManagement(obj client.Object) {
	core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
}

// RemoveNomosLabelsAndAnnotations removes syncer-managed Nomos system
// annotations and labels from the given resource. The resource is modified in
// place. Returns true if the object was modified.
func RemoveNomosLabelsAndAnnotations(obj client.Object) bool {
	before := len(obj.GetAnnotations()) + len(obj.GetLabels())
	core.RemoveAnnotations(obj, append(v1.SyncerAnnotations(), hnc.AnnotationKeyV1A2)...)
	core.RemoveLabels(obj, v1.SyncerLabels())
	after := len(obj.GetAnnotations()) + len(obj.GetLabels())
	return before != after
}
