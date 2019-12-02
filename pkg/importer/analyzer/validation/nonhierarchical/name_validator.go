package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/util/validation"
)

// NameValidator adapts metadata.NameValidator logic for non-hierarchical file structures.
var NameValidator = PerObjectValidator(validName)

// validName returns a MultiError if the object has an invalid metadata.name, or nil otherwise.
func validName(o ast.FileObject) status.Error {
	if o.GetName() == "" {
		// Name MUST NOT be empty
		return MissingObjectNameError(&o)
	}

	errs := validation.IsDNS1123Subdomain(o.GetName())
	if errs != nil {
		return InvalidMetadataNameError(&o)
	}
	return nil
}

// MissingObjectNameErrorCode is the error code for MissingObjectNameError
const MissingObjectNameErrorCode = "1031"

var missingObjectNameError = status.NewErrorBuilder(MissingObjectNameErrorCode)

// MissingObjectNameError reports that an object has no name.
func MissingObjectNameError(resource id.Resource) status.Error {
	return missingObjectNameError.
		Sprintf("Configs must declare `metadata.name`:").
		BuildWithResources(resource)
}

// InvalidMetadataNameErrorCode is the error code for InvalidMetadataNameError
const InvalidMetadataNameErrorCode = "1036"

var invalidMetadataNameError = status.NewErrorBuilder(InvalidMetadataNameErrorCode)

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
func InvalidMetadataNameError(resource id.Resource) status.Error {
	return invalidMetadataNameError.
		Sprintf("Configs MUST define a `metadata.name` that is shorter than 254 characters, consists of lower case alphanumeric " +
			"characters, '-' or '.', and must start and end with an alphanumeric character. Rename or remove the config:").
		BuildWithResources(resource)
}
