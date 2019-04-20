package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedObjectErrorCode is the error code for UnsupportedObjectError
const UnsupportedObjectErrorCode = "1043"

func init() {
	status.Register(UnsupportedObjectErrorCode, UnsupportedObjectError{
		Resource: role(),
	})
}

// UnsupportedObjectError reports than an unsupported object is in the namespaces/ sub-directories or clusters/ directory.
type UnsupportedObjectError struct {
	id.Resource
}

var _ status.ResourceError = UnsupportedObjectError{}

// Error implements error.
func (e UnsupportedObjectError) Error() string {
	return status.Format(e,
		"%s cannot configure this resource. To fix, remove this resource from the repo.",
		configmanagement.ProductName)
}

// Code implements Error
func (e UnsupportedObjectError) Code() string { return UnsupportedObjectErrorCode }

// Resources implements ResourceError
func (e UnsupportedObjectError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e UnsupportedObjectError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
