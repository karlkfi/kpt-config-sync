package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedObjectErrorCode is the error code for UnsupportedObjectError
const UnsupportedObjectErrorCode = "1043"

func init() {
	status.AddExamples(UnsupportedObjectErrorCode, UnsupportedObjectError(role()))
}

var unsupportedObjectError = status.NewErrorBuilder(UnsupportedObjectErrorCode)

// UnsupportedObjectError reports than an unsupported object is in the namespaces/ sub-directories or clusters/ directory.
func UnsupportedObjectError(resource id.Resource) status.Error {
	return unsupportedObjectError.WithResources(resource).Errorf(
		"%s cannot configure this resource. To fix, remove this resource from the repo.",
		configmanagement.ProductName)
}
