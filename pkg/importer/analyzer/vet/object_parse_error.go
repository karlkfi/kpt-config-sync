package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// ObjectParseErrorCode is the code for ObjectParseError.
const ObjectParseErrorCode = "1006"

func init() {
	status.Register(ObjectParseErrorCode, ObjectParseError{
		Resource: role(),
	})
}

// ObjectParseError reports that an object of known type did not match its definition, and so it was
// read in as an *unstructured.Unstructured.
type ObjectParseError struct {
	id.Resource
}

var _ status.ResourceError = ObjectParseError{}

// Error implements error.
func (e ObjectParseError) Error() string {
	return status.Format(e, "The following config is not parseable as a %v:", e.GroupVersionKind())
}

// Code implements Error.
func (e ObjectParseError) Code() string { return ObjectParseErrorCode }

// Resources implements ResourceError.
func (e ObjectParseError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e ObjectParseError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
