package vet

import (
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// UnsupportedCRDRemovalErrorCode is the error code for UnsupportedCRDRemovalError
const UnsupportedCRDRemovalErrorCode = "1047"

func init() {
	status.Register(UnsupportedCRDRemovalErrorCode, UnsupportedCRDRemovalError{customResourceDefinition()})
}

// UnsupportedCRDRemovalError reports than a CRD was removed, but its corresponding CRs weren't.
type UnsupportedCRDRemovalError struct {
	Resource id.Resource
}

var _ status.ResourceError = UnsupportedCRDRemovalError{}

// Error implements error.
func (e UnsupportedCRDRemovalError) Error() string {
	return status.Format(e,
		"Removing a a CRD and leaving the corresponding Custom Resources in the repo is disallowed. To fix, "+
			"remove the CRD along with the Custom Resources.")
}

// Code implements Error
func (e UnsupportedCRDRemovalError) Code() string { return UnsupportedCRDRemovalErrorCode }

// Resources implements ResourceError
func (e UnsupportedCRDRemovalError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e UnsupportedCRDRemovalError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
