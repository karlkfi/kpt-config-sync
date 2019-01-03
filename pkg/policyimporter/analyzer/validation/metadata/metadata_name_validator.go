package metadata

import (
	"path"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
)

// MetadataNameValidator validates the value of metadata.name
var MetadataNameValidator = &syntax.FileObjectValidator{
	ValidateFn: func(fileObject ast.FileObject) error {
		gvk := fileObject.GroupVersionKind()

		if fileObject.Name() == "" {
			// Name MUST NOT be empty
			return vet.MissingObjectNameError{ResourceID: &fileObject}
		} else if isDefaultCrdAllowedInNomos(gvk) {
			// If CRD, then namee must be a valid DNS1123 subdomain
			errs := utilvalidation.IsDNS1123Subdomain(fileObject.Name())
			if errs != nil {
				return vet.InvalidMetadataNameError{ResourceID: &fileObject}
			}
		} else if gvk == kinds.Namespace() {
			expectedName := path.Base(path.Dir(fileObject.Source()))
			if expectedName == repo.NamespacesDir {
				return vet.IllegalTopLevelNamespaceError{ResourceID: &fileObject}
			}
			if fileObject.Name() != expectedName {
				return vet.InvalidNamespaceNameError{ResourceID: &fileObject, Expected: expectedName}
			}
		}
		return nil
	},
}

// isDefaultCrdAllowedInNomos checks if a Resource is a CRD that comes with a default Nomos installation.
//
// This explicitly does not check for Nomos or Application even though they are CRDs because they
// should never be in a Nomos repository anyway.
func isDefaultCrdAllowedInNomos(gvk schema.GroupVersionKind) bool {
	return gvk.Group == policyhierarchy.GroupName || (gvk == kinds.Cluster())
}
