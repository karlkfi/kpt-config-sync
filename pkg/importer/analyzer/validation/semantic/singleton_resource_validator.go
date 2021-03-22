package semantic

import (
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(ekitson): Replace usage of this error with id.MultipleSingletonsError instead

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "1030"

var multipleSingletonsError = status.NewErrorBuilder(MultipleSingletonsErrorCode)

// MultipleSingletonsError reports that multiple singletons are defined in the same directory.
func MultipleSingletonsError(duplicates ...client.Object) status.Error {
	var gvk schema.GroupVersionKind
	if len(duplicates) > 0 {
		gvk = duplicates[0].GetObjectKind().GroupVersionKind()
	}

	return multipleSingletonsError.
		Sprintf("Multiple %v resources cannot exist in the same directory. "+
			"To fix, remove the duplicate config(s) such that no more than 1 remains:", gvk.GroupKind().String()).
		BuildWithResources(duplicates...)
}
