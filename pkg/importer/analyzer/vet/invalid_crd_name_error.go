package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidCRDNameErrorCode is the error code for InvalidCRDNameError.
const InvalidCRDNameErrorCode = "1048"

func init() {
	status.Register(InvalidCRDNameErrorCode, InvalidCRDNameError{
		Resource: customResourceDefinition(),
	})
}

// InvalidCRDNameError reports a CRD with an invalid name in the repo.
type InvalidCRDNameError struct {
	id.Resource
}

var _ status.ResourceError = InvalidCRDNameError{}

// Error implements error.
func (e InvalidCRDNameError) Error() string {
	return status.Format(e,
		"The CustomResourceDefinition has an invalid name. To fix, change the name to `spec.names.plural+\".\"+spec.group`.")
}

// Code implements Error
func (e InvalidCRDNameError) Code() string { return InvalidCRDNameErrorCode }

// Resources implements ResourceError
func (e InvalidCRDNameError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e InvalidCRDNameError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
