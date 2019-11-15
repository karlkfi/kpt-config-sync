package metadata

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
)

// NewNameValidator validates the value of metadata.name
func NewNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		err := ValidName(o)
		if err != nil {
			return err
		}
		if o.GroupVersionKind() == kinds.Namespace() {
			// TODO(willbeason) Move this to its own Validator.
			expectedName := o.Dir().Base()
			if expectedName == repo.NamespacesDir {
				return IllegalTopLevelNamespaceError(&o)
			}
			if o.GetName() != expectedName {
				return InvalidNamespaceNameError(&o, expectedName)
			}
		}
		return nil
	})
}

// ValidName returns a MultiError if the object has an invalid metadata.name, or nil otherwise.
func ValidName(o ast.FileObject) status.Error {
	gvk := o.GroupVersionKind()

	if o.GetName() == "" {
		// Name MUST NOT be empty
		return MissingObjectNameError(&o)
	} else if isDefaultCrdAllowedInNomos(gvk) {
		// If CRD, then name must be a valid DNS1123 subdomain
		errs := validation.IsDNS1123Subdomain(o.GetName())
		if errs != nil {
			return InvalidMetadataNameError(&o)
		}
	}
	return nil
}

// isDefaultCrdAllowedInNomos checks if a Resource is a CRD that comes with a default Nomos installation.
//
// This explicitly does not check for Nomos or Application even though they are CRDs because they
// should never be in a Nomos repository anyway.
func isDefaultCrdAllowedInNomos(gvk schema.GroupVersionKind) bool {
	return gvk.Group == configmanagement.GroupName || (gvk == kinds.Cluster())
}

// IllegalTopLevelNamespaceErrorCode is the error code for IllegalTopLevelNamespaceError
const IllegalTopLevelNamespaceErrorCode = "1019"

var illegalTopLevelNamespaceError = status.NewErrorBuilder(IllegalTopLevelNamespaceErrorCode)

// IllegalTopLevelNamespaceError reports that there may not be a Namespace declared directly in namespaces/
// Error implements error
func IllegalTopLevelNamespaceError(resource id.Resource) status.Error {
	return illegalTopLevelNamespaceError.
		Sprintf("%[2]ss MUST be declared in subdirectories of %[1]s/. Create a subdirectory for %[2]ss declared in:",
			repo.NamespacesDir, node.Namespace).
		BuildWithResources(resource)
}

// InvalidNamespaceNameErrorCode is the error code for InvalidNamespaceNameError
const InvalidNamespaceNameErrorCode = "1020"

var invalidNamespaceNameErrorstatus = status.NewErrorBuilder(InvalidNamespaceNameErrorCode)

// InvalidNamespaceNameError reports that a Namespace has an invalid name.
func InvalidNamespaceNameError(resource id.Resource, expected string) status.Error {
	return invalidNamespaceNameErrorstatus.
		Sprintf("A %[1]s MUST declare `metadata.name` that matches the name of its directory.\n\n"+
			"expected metadata.name: %[2]s",
			node.Namespace, expected).
		BuildWithResources(resource)
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
