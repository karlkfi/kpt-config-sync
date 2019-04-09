package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalFieldsInConfigErrorCode is the error code for IllegalFieldsInConfigError
const IllegalFieldsInConfigErrorCode = "1045"

func init() {
	status.Register(IllegalFieldsInConfigErrorCode, IllegalFieldsInConfigError{
		Resource: replicaSet(),
		Field:    id.OwnerReference,
	})
}

// IllegalFieldsInConfigError reports that an object has an illegal field set.
type IllegalFieldsInConfigError struct {
	id.Resource
	Field id.DisallowedField
}

var _ status.ResourceError = &IllegalFieldsInConfigError{}

// Error implements error
func (e IllegalFieldsInConfigError) Error() string {
	return status.Format(e,
		"Configs with %[1]q specified are not allowed. "+
			"To fix, either remove the config or remove the %[1]q field in the config:",
		e.Field)
}

// Code implements Error
func (e IllegalFieldsInConfigError) Code() string {
	return IllegalFieldsInConfigErrorCode
}

// Resources implements ResourceError
func (e IllegalFieldsInConfigError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
