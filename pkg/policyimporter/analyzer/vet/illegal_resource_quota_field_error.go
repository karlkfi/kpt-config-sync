package vet

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/status"
	"k8s.io/api/core/v1"
)

// IllegalResourceQuotaFieldErrorCode is the error code for llegalResourceQuotaFieldError
const IllegalResourceQuotaFieldErrorCode = "1008"

func init() {
	register(IllegalResourceQuotaFieldErrorCode)
}

// IllegalResourceQuotaFieldError represents illegal fields set on ResourceQuota objects.
type IllegalResourceQuotaFieldError struct {
	// Path is the repository directory where the quota is located.
	Path nomospath.Path
	// ResourceQuota is the quota with the illegal field.
	ResourceQuota v1.ResourceQuota
	// Field is the illegal field set.
	Field string
}

// Error implements error.
func (e IllegalResourceQuotaFieldError) Error() string {
	return status.Format(e,
		"ResourceQuota objects MUST NOT set scope when hierarchyMode is set to hierarchicalQuota. "+
			"Remove illegal field %[1]s from object %[2]s located at directory %[3]q.",
		e.Field, e.ResourceQuota.GetObjectMeta().GetName(), e.Path.SlashPath())
}

// Code implements Error
func (e IllegalResourceQuotaFieldError) Code() string { return IllegalResourceQuotaFieldErrorCode }
