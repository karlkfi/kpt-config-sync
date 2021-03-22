package nonhierarchical

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalHierarchicalKindErrorCode is the error code for illegalHierarchicalKindErrors.
const IllegalHierarchicalKindErrorCode = "1032"

var illegalHierarchicalKindError = status.NewErrorBuilder(IllegalHierarchicalKindErrorCode)

// IllegalHierarchicalKind reports that a type is not permitted if hierarchical parsing is disabled.
func IllegalHierarchicalKind(resource client.Object) status.Error {
	return illegalHierarchicalKindError.
		Sprintf("The type %v is not allowed if `sourceFormat` is set to "+
			"`unstructured`. To fix, remove the problematic config, or convert your repo "+
			"to use `sourceFormat: hierarchy`.", resource.GetObjectKind().GroupVersionKind().GroupKind().String()).
		BuildWithResources(resource)
}
