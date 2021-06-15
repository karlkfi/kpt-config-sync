package reconcile

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncedAt marks the resource as synced at the passed sync token.
func SyncedAt(obj client.Object, token string) {
	core.SetAnnotation(obj, v1.SyncTokenAnnotationKey, token)
}

// enableManagement marks the resource as Nomos-managed.
func enableManagement(obj client.Object) {
	core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementEnabled)
	core.SetAnnotation(obj, constants.ResourceIDKey, core.GKNN(obj))
	core.SetLabel(obj, v1.ManagedByKey, v1.ManagedByValue)
}
