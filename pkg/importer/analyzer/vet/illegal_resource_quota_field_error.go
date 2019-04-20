package vet

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
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

var _ status.ResourceError = IllegalResourceQuotaFieldError{}

// IllegalResourceQuotaFieldError represents illegal fields set on ResourceQuota objects.
type IllegalResourceQuotaFieldError struct {
	Resource id.Resource
	// Field is the illegal field set.
	Field string
}

// Error implements error.
func (e IllegalResourceQuotaFieldError) Error() string {
	return status.Format(e,
		"A ResourceQuota config MUST NOT set scope when hierarchyMode is set to hierarchicalQuota. "+
			"Remove illegal field %s from:",
		e.Field)
}

// Code implements Error
func (e IllegalResourceQuotaFieldError) Code() string { return IllegalResourceQuotaFieldErrorCode }

// Resources implements ResourceError
func (e IllegalResourceQuotaFieldError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalResourceQuotaFieldError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
