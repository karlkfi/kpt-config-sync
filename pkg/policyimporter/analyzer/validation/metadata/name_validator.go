package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
)

// NewNameValidator validates the value of metadata.name
func NewNameValidator() *visitor.ValidatorVisitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) *status.MultiError {
			gvk := o.GroupVersionKind()

			if o.Name() == "" {
				// Name MUST NOT be empty
				return status.From(vet.MissingObjectNameError{Resource: &o})
			} else if isDefaultCrdAllowedInNomos(gvk) {
				// If CRD, then name must be a valid DNS1123 subdomain
				errs := validation.IsDNS1123Subdomain(o.Name())
				if errs != nil {
					return status.From(vet.InvalidMetadataNameError{Resource: &o})
				}
			} else if gvk == kinds.Namespace() {
				// TODO(willbeason) Move this to its own Validator.
				expectedName := o.Dir().Base()
				if expectedName == repo.NamespacesDir {
					return status.From(vet.IllegalTopLevelNamespaceError{Resource: &o})
				}
				if o.Name() != expectedName {
					return status.From(vet.InvalidNamespaceNameError{Resource: &o, Expected: expectedName})
				}
			}
			return nil
		})
}

// isDefaultCrdAllowedInNomos checks if a Resource is a CRD that comes with a default Nomos installation.
//
// This explicitly does not check for Nomos or Application even though they are CRDs because they
// should never be in a Nomos repository anyway.
func isDefaultCrdAllowedInNomos(gvk schema.GroupVersionKind) bool {
	return gvk.Group == policyhierarchy.GroupName || (gvk == kinds.Cluster())
}
