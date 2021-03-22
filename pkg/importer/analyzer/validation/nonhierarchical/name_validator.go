package nonhierarchical

import (
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MissingObjectNameErrorCode is the error code for MissingObjectNameError
const MissingObjectNameErrorCode = "1031"

var missingObjectNameError = status.NewErrorBuilder(MissingObjectNameErrorCode)

// MissingObjectNameError reports that an object has no name.
func MissingObjectNameError(resource client.Object) status.Error {
	return missingObjectNameError.
		Sprintf("Configs must declare `metadata.name`:").
		BuildWithResources(resource)
}

// InvalidMetadataNameErrorCode is the error code for InvalidMetadataNameError
const InvalidMetadataNameErrorCode = "1036"

var invalidMetadataNameError = status.NewErrorBuilder(InvalidMetadataNameErrorCode)

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
func InvalidMetadataNameError(resource client.Object) status.Error {
	return invalidMetadataNameError.
		Sprintf("Configs MUST define a `metadata.name` that is shorter than 254 characters, consists of lower case alphanumeric " +
			"characters, '-' or '.', and must start and end with an alphanumeric character. Rename or remove the config:").
		BuildWithResources(resource)
}
