package id

import (
	"github.com/google/nomos/pkg/status"
)

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "2012"

func init() {
	// TODO: add a way to generate valid error without dependency cycle.
	//status.Register(MultipleSingletonsErrorCode, MultipleSingletonsError{})
}

// MultipleSingletonsError reports that multiple singleton resources were found on the cluster.
type MultipleSingletonsError struct {
	Duplicates []Resource
}

var _ ResourceError = &MultipleSingletonsError{}

// Error implements error
func (e MultipleSingletonsError) Error() string {
	return status.Format(e, "Found more than one %[1]s:\n%[2]s", resourceName(e.Duplicates), FormatResources(e))
}

// Code implements Error
func (e MultipleSingletonsError) Code() string {
	return MultipleSingletonsErrorCode
}

// Resources implements ResourceError
func (e MultipleSingletonsError) Resources() []Resource {
	return e.Duplicates
}

// MultipleSingletonsWrap returns a MultipleSingletonsError wrapping the given Resources.
func MultipleSingletonsWrap(resources ...Resource) MultipleSingletonsError {
	return MultipleSingletonsError{Duplicates: resources}
}

func resourceName(dups []Resource) string {
	if len(dups) == 0 {
		return "singleton"
	}
	return dups[0].Name()
}
