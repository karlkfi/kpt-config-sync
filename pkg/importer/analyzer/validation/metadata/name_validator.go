package metadata

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
)

// InvalidMetadataNameErrorCode is the error code for InvalidMetadataNameError
const InvalidMetadataNameErrorCode = "1036"

func init() {
	r := fake.Role()
	r.MetaObject().SetName("a`b.c")
	status.AddExamples(InvalidMetadataNameErrorCode, InvalidMetadataNameError(&r))
}

var invalidMetadataNameError = status.NewErrorBuilder(InvalidMetadataNameErrorCode)

// InvalidMetadataNameError represents the usage of a non-RFC1123 compliant metadata.name
func InvalidMetadataNameError(resource id.Resource) status.Error {
	return invalidMetadataNameError.WithResources(resource).Errorf(
		"Configs MUST define a `metadata.name` that is shorter than 254 characters, consists of lower case alphanumeric " +
			"characters, '-' or '.', and must start and end with an alphanumeric character. Rename or remove the config:")
}

// NewNameValidator validates the value of metadata.name
func NewNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(func(o ast.FileObject) status.MultiError {
		err := status.From(ValidName(o))
		if err != nil {
			return err
		}
		if o.GroupVersionKind() == kinds.Namespace() {
			// TODO(willbeason) Move this to its own Validator.
			expectedName := o.Dir().Base()
			if expectedName == repo.NamespacesDir {
				return status.From(vet.IllegalTopLevelNamespaceError(&o))
			}
			if o.Name() != expectedName {
				return status.From(vet.InvalidNamespaceNameError(&o, expectedName))
			}
		}
		return nil
	})
}

// ValidName returns a MultiError if the object has an invalid metadata.name, or nil otherwise.
func ValidName(o ast.FileObject) status.Error {
	gvk := o.GroupVersionKind()

	if o.Name() == "" {
		// Name MUST NOT be empty
		return vet.MissingObjectNameError(&o)
	} else if isDefaultCrdAllowedInNomos(gvk) {
		// If CRD, then name must be a valid DNS1123 subdomain
		errs := validation.IsDNS1123Subdomain(o.Name())
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
