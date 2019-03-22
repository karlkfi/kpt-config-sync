package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// InvalidMetadataNameErrorCode is the error code for InvalidMetadataNameError
const InvalidMetadataNameErrorCode = "1036"

func init() {
	r := role()
	r.MetaObject().SetName("a`b.c")
	status.Register(InvalidMetadataNameErrorCode, InvalidMetadataNameError{
		Resource: r,
	})
}

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
type InvalidMetadataNameError struct {
	id.Resource
}

var _ id.ResourceError = &InvalidMetadataNameError{}

// Error implements error.
func (e InvalidMetadataNameError) Error() string {
	return status.Format(e,
		"Resources MUST define a metadata.name which is a valid RFC1123 DNS subdomain. Rename or remove the Resource:\n\n"+
			"%[1]s",
		id.PrintResource(e))
}

// Code implements Error
func (e InvalidMetadataNameError) Code() string { return InvalidMetadataNameErrorCode }

// Resources implements ResourceError
func (e InvalidMetadataNameError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
