package vet

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalFieldsInConfigErrorCode is the error code for IllegalFieldsInConfigError
const IllegalFieldsInConfigErrorCode = "1045"

func init() {
	status.AddExamples(IllegalFieldsInConfigErrorCode,
		IllegalFieldsInConfigError(replicaSet(), id.OwnerReference))
}

var illegalFieldsInConfigError = status.NewErrorBuilder(IllegalFieldsInConfigErrorCode)

// IllegalFieldsInConfigError reports that an object has an illegal field set.
func IllegalFieldsInConfigError(resource id.Resource, field id.DisallowedField) status.Error {
	return illegalFieldsInConfigError.WithResources(resource).Errorf(
		"Configs with %[1]q specified are not allowed. "+
			"To fix, either remove the config or remove the %[1]q field in the config:",
		field)
}
