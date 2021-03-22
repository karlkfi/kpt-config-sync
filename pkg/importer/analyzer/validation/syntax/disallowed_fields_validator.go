package syntax

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalFieldsInConfigErrorCode is the error code for IllegalFieldsInConfigError
const IllegalFieldsInConfigErrorCode = "1045"

var illegalFieldsInConfigErrorBuilder = status.NewErrorBuilder(IllegalFieldsInConfigErrorCode)

// IllegalFieldsInConfigError reports that an object has an illegal field set.
func IllegalFieldsInConfigError(resource client.Object, field id.DisallowedField) status.Error {
	return illegalFieldsInConfigErrorBuilder.
		Sprintf("Configs with %[1]q specified are not allowed. "+
			"To fix, either remove the config or remove the %[1]q field in the config:",
			field).
		BuildWithResources(resource)
}
