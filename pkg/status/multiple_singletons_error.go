package status

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

// MultipleSingletonsErrorCode is the error code for MultipleSingletonsError
const MultipleSingletonsErrorCode = "2012"

func init() {
	Register(MultipleSingletonsErrorCode, MultipleSingletonsError{})
}

// MultipleSingletonsError reports that multiple singleton resources were found on the cluster.
type MultipleSingletonsError struct {
	Duplicates []id.Resource
}

var _ ResourceError = MultipleSingletonsError{}

// Error implements error
func (e MultipleSingletonsError) Error() string {
	return Format(e, "Found more than one %[1]s:\n%[2]s", resourceName(e.Duplicates), formatResources(e.Resources()))
}

// Code implements Error
func (e MultipleSingletonsError) Code() string {
	return MultipleSingletonsErrorCode
}

// Resources implements ResourceError
func (e MultipleSingletonsError) Resources() []id.Resource {
	return e.Duplicates
}

// ToCME implements ToCMEr.
func (e MultipleSingletonsError) ToCME() v1.ConfigManagementError {
	return FromResourceError(e)
}

// MultipleSingletonsWrap returns a MultipleSingletonsError wrapping the given Resources.
func MultipleSingletonsWrap(resources ...id.Resource) MultipleSingletonsError {
	return MultipleSingletonsError{resources}
}

func resourceName(dups []id.Resource) string {
	if len(dups) == 0 {
		return "singleton"
	}
	return dups[0].Name()
}
