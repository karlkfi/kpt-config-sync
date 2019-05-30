package status

import (
	"github.com/google/nomos/pkg/importer/id"
)

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "2012"

func init() {
	AddExamples(MultipleSingletonsErrorCode, MultipleSingletonsError())
}

var multipleSingletonsError = NewErrorBuilder(MultipleSingletonsErrorCode)

// MultipleSingletonsError reports that multiple singleton resources were found on the cluster.
func MultipleSingletonsError(duplicates ...id.Resource) Error {
	return multipleSingletonsError.WithResources(duplicates...).Errorf(
		"Found more than one %[1]s:\n%[2]s", resourceName(duplicates), formatResources(duplicates))
}

func resourceName(dups []id.Resource) string {
	if len(dups) == 0 {
		return "singleton"
	}
	return dups[0].Name()
}
