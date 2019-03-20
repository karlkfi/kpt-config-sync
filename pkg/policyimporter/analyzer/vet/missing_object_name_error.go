package vet

import (
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// MissingObjectNameErrorCode is the error code for MissingObjectNameError
const MissingObjectNameErrorCode = "1031"

func init() {
	status.Register(MissingObjectNameErrorCode, MissingObjectNameError{})
}

// MissingObjectNameError reports that an object has no name.
type MissingObjectNameError struct {
	id.Resource
}

var _ id.ResourceError = &MissingObjectNameError{}

// Error implements error
func (e MissingObjectNameError) Error() string {
	return status.Format(e,
		"Resources must declare metadata.name:\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e MissingObjectNameError) Code() string { return MissingObjectNameErrorCode }

// Resources implements ResourceError
func (e MissingObjectNameError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
