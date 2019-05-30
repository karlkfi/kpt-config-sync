package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalResourceQuotaFieldErrorCode is the error code for llegalResourceQuotaFieldError
const IllegalResourceQuotaFieldErrorCode = "1008"

func init() {
	status.AddExamples(IllegalResourceQuotaFieldErrorCode, IllegalResourceQuotaFieldError(
		resourceQuota(),
		"scopes",
	))
}

var illegalResourceQuotaFieldError = status.NewErrorBuilder(IllegalResourceQuotaFieldErrorCode)

// IllegalResourceQuotaFieldError represents illegal fields set on ResourceQuota objects.
func IllegalResourceQuotaFieldError(resource id.Resource, field string) status.Error {
	return illegalResourceQuotaFieldError.WithResources(resource).Errorf(
		"A ResourceQuota config MUST NOT set scope when hierarchyMode is set to hierarchicalQuota. "+
			"Remove illegal field %s from:",
		field)
}
