package status

import "sigs.k8s.io/controller-runtime/pkg/client"

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "2012"

var multipleSingletonsError = NewErrorBuilder(MultipleSingletonsErrorCode)

// MultipleSingletonsError reports that multiple singleton resources were found on the cluster.
func MultipleSingletonsError(duplicates ...client.Object) Error {
	return multipleSingletonsError.Sprintf(
		"Unsupported number of %s resource found: %d, want: 1.", resourceName(duplicates), len(duplicates)).BuildWithResources(duplicates...)
}

func resourceName(dups []client.Object) string {
	if len(dups) == 0 {
		return "singleton"
	}
	return dups[0].GetObjectKind().GroupVersionKind().GroupKind().String()
}
