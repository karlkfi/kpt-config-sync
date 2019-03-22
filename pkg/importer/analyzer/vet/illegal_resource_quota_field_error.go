package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalResourceQuotaFieldErrorCode is the error code for llegalResourceQuotaFieldError
const IllegalResourceQuotaFieldErrorCode = "1008"

func init() {
	status.Register(IllegalResourceQuotaFieldErrorCode, IllegalResourceQuotaFieldError{
		Resource: resourceQuota(),
		Field:    "scopes",
	})
}

// IllegalResourceQuotaFieldError represents illegal fields set on ResourceQuota objects.
type IllegalResourceQuotaFieldError struct {
	Resource id.Resource
	// Field is the illegal field set.
	Field string
}

// Error implements error.
func (e IllegalResourceQuotaFieldError) Error() string {
	return status.Format(e,
		"ResourceQuota objects MUST NOT set scope when hierarchyMode is set to hierarchicalQuota. "+
			"Remove illegal field %[1]s from:\n\n%[2]s",
		e.Field, id.PrintResource(e.Resource))
}

// Code implements Error
func (e IllegalResourceQuotaFieldError) Code() string { return IllegalResourceQuotaFieldErrorCode }

// Resources implements ResourceError
func (e IllegalResourceQuotaFieldError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}
