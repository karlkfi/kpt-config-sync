package reconcile

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncedAt marks the resource as synced at the passed sync token.
func SyncedAt(obj client.Object, token string) {
	core.SetAnnotation(obj, metadata.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-managed.
func enableManagement(obj client.Object) {
	core.SetAnnotation(obj, metadata.ResourceManagementKey, metadata.ResourceManagementEnabled)
	core.SetAnnotation(obj, metadata.ResourceIDKey, core.GKNN(obj))
	core.SetLabel(obj, metadata.ManagedByKey, metadata.ManagedByValue)
}
