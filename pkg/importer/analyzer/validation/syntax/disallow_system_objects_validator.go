package syntax

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsSystemOnly returns true if the GVK is only allowed in the system/ directory.
// It returns true iff the object is allowed in system/, but no other directories.
func IsSystemOnly(gvk schema.GroupVersionKind) bool {
	switch gvk {
	case kinds.Repo(), kinds.HierarchyConfig():
		return true
	default:
		return false
	}
}

// IllegalSystemResourcePlacementErrorCode is the error code for IllegalSystemResourcePlacementError
const IllegalSystemResourcePlacementErrorCode = "1033"

var illegalSystemResourcePlacementError = status.NewErrorBuilder(IllegalSystemResourcePlacementErrorCode)

// IllegalSystemResourcePlacementError reports that a configmanagement.gke.io object has been defined outside of system/
func IllegalSystemResourcePlacementError(resource client.Object) status.Error {
	return illegalSystemResourcePlacementError.
		Sprintf("A config of the below kind MUST NOT be declared outside %[1]s/:",
			repo.SystemDir).
		BuildWithResources(resource)
}
