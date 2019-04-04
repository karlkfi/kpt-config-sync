package reconcile

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/object"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// syncedAt marks the resource as synced at the passed sync token.
func syncedAt(obj object.Annotated, token string) {
	object.SetAnnotation(obj, v1.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-manged.
func enableManagement(obj object.Annotated) {
	object.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
}

// enableQuota enables quota management for the resource.
func enableQuota(obj object.Labeled) {
	object.SetLabel(obj, v1.ConfigManagementQuotaKey, v1.ConfigManagementQuotaValue)
}

// removeNomosMeta removes syncer-managed Nomos system annotations and labels from the given resource.
// The resource is modified in place.
func removeNomosMeta(obj metav1.Object) {
	object.RemoveAnnotations(obj, v1.SyncerAnnotations()...)
	object.RemoveLabels(obj, v1.SyncerLabels()...)
}
