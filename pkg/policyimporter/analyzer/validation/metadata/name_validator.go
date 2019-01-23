package metadata

import (
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
)

func init() {
	Register(NameValidatorFactory)
}

// NameValidatorFactory validates the value of metadata.name
var NameValidatorFactory = SyntaxValidatorFactory{
	fn: func(meta ResourceMeta) error {
		gvk := meta.GroupVersionKind()

		if meta.Name() == "" {
			// Name MUST NOT be empty
			return vet.MissingObjectNameError{Resource: meta}
		} else if isDefaultCrdAllowedInNomos(gvk) {
			// If CRD, then name must be a valid DNS1123 subdomain
			errs := validation.IsDNS1123Subdomain(meta.Name())
			if errs != nil {
				return vet.InvalidMetadataNameError{Resource: meta}
			}
		} else if gvk == kinds.Namespace() {
			// TODO(willbeason) Move this to Namespace-specific package.
			expectedName := meta.Dir().Base()
			if expectedName == repo.NamespacesDir {
				return vet.IllegalTopLevelNamespaceError{Resource: meta}
			}
			if meta.Name() != expectedName {
				return vet.InvalidNamespaceNameError{Resource: meta, Expected: expectedName}
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
